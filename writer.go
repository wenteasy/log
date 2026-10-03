package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RollingFileWriter は時刻でファイルを切り替える io.Writer。
// ファイル名は "<prefix>_<時刻>.log"（prefix が空なら "<時刻>.log"）。
// 書くたびに時刻からファイル名を決め、変わっていたら前のファイルを閉じて次を開く。
// 並行に書いてよい。
//
// 既にあるファイルを開いたときは、続きから書く。そのとき区切りの行を 1 行入れ、
// 閉じるときにも 1 行入れる（DisableMarkers で止められる。JSON で書くときなど）。
type RollingFileWriter struct {
	mu     sync.Mutex
	dir    string
	prefix string
	format string
	marker bool
	now    func() time.Time
	name   string // 開いているファイルの名前（パスではない）
	target *os.File
}

// Interval はファイルを切り替える間隔。
type Interval int

const (
	Second Interval = iota
	Minute
	Hour
	Day
	Month
	Year
	None // 切り替えない。1 つのファイルに書き続ける（ファイル名は prefix。空なら "log.log"）
)

func (i Interval) format() string {
	switch i {
	case Second:
		return "20060102150405"
	case Minute:
		return "200601021504"
	case Hour:
		return "2006010215"
	case Day:
		return "20060102"
	case Month:
		return "200601"
	case Year:
		return "2006"
	}
	return ""
}

// NewRollingFileWriter は dir へ書く RollingFileWriter を返す。dir が無ければ作る。
// ファイルは最初に書いたときに開く。
func NewRollingFileWriter(dir string, i Interval) (*RollingFileWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}
	return &RollingFileWriter{dir: dir, format: i.format(), marker: true, now: time.Now}, nil
}

// SetPrefix はファイル名の頭に付ける文字列を決める（"app" → "app_20260102.log"）。
// 次に書いたときから効く。
func (w *RollingFileWriter) SetPrefix(p string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.prefix = p
}

// SetFormat は時刻の書式（time.Format の書き方）を直接決める。Interval の既定の書式の代わり。
// 切り替わる間隔は書式で決まる（書式に秒が無ければ、秒ごとには切り替わらない）。
func (w *RollingFileWriter) SetFormat(f string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.format = f
}

// DisableMarkers は、続きから書くときと閉じるときの区切りの行を書かないようにする。
func (w *RollingFileWriter) DisableMarkers() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.marker = false
}

// Path は今開いているファイルのパスを返す。まだ 1 度も書いていなければ空。
func (w *RollingFileWriter) Path() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.target == nil {
		return ""
	}
	return w.target.Name()
}

const (
	reopenMarker = "--------------> since the file exists, starting from here.\n"
	closeMarker  = "<-------------- end here.\n"
)

func (w *RollingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	name := w.fileName()
	if w.target == nil || name != w.name {
		if err := w.open(name); err != nil {
			return 0, err
		}
	}
	return w.target.Write(p)
}

// Close は開いているファイルを閉じる。何度呼んでもよい。閉じたあとに書けば、また開く。
func (w *RollingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.close()
}

func (w *RollingFileWriter) close() error {
	if w.target == nil {
		return nil
	}
	if w.marker {
		w.target.WriteString(closeMarker)
	}
	err := w.target.Close()
	w.target = nil
	w.name = ""
	return err
}

func (w *RollingFileWriter) fileName() string {
	var parts []string
	if w.prefix != "" {
		parts = append(parts, w.prefix)
	}
	if w.format != "" {
		parts = append(parts, w.now().Format(w.format))
	}
	if len(parts) == 0 {
		return "log.log"
	}
	return strings.Join(parts, "_") + ".log"
}

func (w *RollingFileWriter) open(name string) error {
	w.close()
	path := filepath.Join(w.dir, name)
	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile() error: %w", err)
	}
	if statErr == nil && w.marker {
		f.WriteString("\n" + reopenMarker)
	}
	w.target = f
	w.name = name
	return nil
}
