package log

import "time"

// SetNow はテストで時刻を差し替える。
func (w *RollingFileWriter) SetNow(now func() time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.now = now
}
