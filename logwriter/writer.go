package logwriter

import (
	"log"
	"sync"
	"time"

	"github.com/aghiadodeh/go-monitoring/models"
	"gorm.io/gorm"
)

// Writer is a high-performance async batch writer for request logs.
// It receives log entries via a buffered channel and flushes them
// to the database in batches, minimizing per-request overhead.
type Writer struct {
	db            *gorm.DB
	ch            chan models.RequestLog
	batchSize     int
	flushInterval time.Duration
	done          chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	closed        bool
	once          sync.Once
}

// Options configures the Writer.
type Options struct {
	BufferSize    int           // channel capacity          (default: 10 000)
	BatchSize     int           // records per INSERT        (default: 100)
	FlushInterval time.Duration // max idle time before flush (default: 5 s)
	Workers       int           // parallel writer goroutines (default: 1)
}

// New creates a Writer and starts its background worker(s).
func New(db *gorm.DB, opts Options) *Writer {
	if opts.BufferSize <= 0 {
		opts.BufferSize = 10_000
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 5 * time.Second
	}
	if opts.Workers <= 0 {
		opts.Workers = 1
	}

	w := &Writer{
		db:            db,
		ch:            make(chan models.RequestLog, opts.BufferSize),
		batchSize:     opts.BatchSize,
		flushInterval: opts.FlushInterval,
		done:          make(chan struct{}),
	}

	for i := 0; i < opts.Workers; i++ {
		w.wg.Add(1)
		go w.worker()
	}

	return w
}

// Write enqueues a log entry. It never blocks the caller: if the
// buffer is full or the writer has been shut down, the entry is
// silently dropped.
func (w *Writer) Write(entry models.RequestLog) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return
	}

	select {
	case w.ch <- entry:
	default:
		// Buffer full – drop to protect request latency.
		log.Println("[go-monitoring] warning: log buffer full, dropping entry")
	}
}

// Shutdown closes the channel and waits for all pending entries
// to be flushed. It is safe to call multiple times.
func (w *Writer) Shutdown() {
	w.once.Do(func() {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()

		close(w.ch)
		w.wg.Wait()
		close(w.done)
	})
}

// Done returns a channel that is closed after Shutdown completes.
func (w *Writer) Done() <-chan struct{} {
	return w.done
}

// worker reads from the channel, accumulates a batch, and flushes
// either when the batch is full or when the flush interval fires.
func (w *Writer) worker() {
	defer w.wg.Done()

	batch := make([]models.RequestLog, 0, w.batchSize)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-w.ch:
			if !ok {
				// Channel closed – flush remaining and exit.
				if len(batch) > 0 {
					w.flush(batch)
				}
				return
			}
			batch = append(batch, entry)
			if len(batch) >= w.batchSize {
				w.flush(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

// flush performs a single multi-row INSERT for the batch.
func (w *Writer) flush(batch []models.RequestLog) {
	if err := w.db.Create(&batch).Error; err != nil {
		log.Printf("[go-monitoring] error flushing %d log(s): %v\n", len(batch), err)
	}
}
