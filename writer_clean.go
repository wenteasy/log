package log

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RollingFileWriter の古いファイルの片付け（圧縮・削除）。
//
// 片付けるのは、このライターのファイル名の形（"<prefix>_<時刻>.log" と ".log.gz"）に
// ぴったり合うものだけ。古さはファイル名の時刻で決める（更新日時は見ない）。
// 今開いているファイルには触らない。
//
// ファイルを開いたとき（最初に書いたとき・切り替えたとき）に、別の goroutine で動く。
// Write は待たない。Close も待たない（途中で終わった圧縮は、次の片付けでやり直す）。

const (
	gzSuffix  = ".gz"
	tmpSuffix = ".gz.tmp" // 圧縮の途中のファイル
)

// SetCompressAfter は、n 間隔より前のファイルを gzip（"<名前>.log.gz"）にする。0 以下なら圧縮しない（既定）。
// 間隔は NewRollingFileWriter に渡した Interval（Day なら日。None では何もしない）。
// n=1 なら、切り替えた時点で前のファイルを圧縮する。次にファイルを開いたときから効く。
func (w *RollingFileWriter) SetCompressAfter(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.compressAfter = n
}

// SetKeep は、n 間隔より前のファイル（.log も .log.gz も）を消す。0 以下なら消さない（既定）。
// 間隔の数え方は SetCompressAfter と同じ。Keep(90) なら、今日を含めて 90 日分が残る。
func (w *RollingFileWriter) SetKeep(n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.keep = n
}

// SetErrorHandler は、片付けで起きたエラーを受け取る関数を決める。nil なら捨てる（既定）。
// f は w のロックを持たずに呼ぶので、f から w へ（slog 経由で）書いてよい。
func (w *RollingFileWriter) SetErrorHandler(f func(error)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onError = f
}

// startCleanLocked は片付けを始める。動いている最中なら、終わったあとにもう一度回す。w.mu を持って呼ぶ。
func (w *RollingFileWriter) startCleanLocked() {
	if w.compressAfter <= 0 && w.keep <= 0 || w.interval == None || w.format == "" {
		return
	}
	if w.cleaning {
		w.again = true
		return
	}
	w.cleaning = true
	w.cleanWG.Add(1)
	go w.cleanLoop()
}

func (w *RollingFileWriter) cleanLoop() {
	defer w.cleanWG.Done()
	for {
		w.mu.Lock()
		j := cleanJob{
			dir: w.dir, prefix: w.prefix, format: w.format, interval: w.interval,
			now: w.now(), current: w.name,
			compressAfter: w.compressAfter, keep: w.keep, onError: w.onError,
		}
		w.again = false
		w.mu.Unlock()

		j.run()

		w.mu.Lock()
		if !w.again {
			w.cleaning = false
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
	}
}

// cleanJob は片付け 1 回分。始めた時点の設定の写しで動く（w のロックを持たない）。
type cleanJob struct {
	dir, prefix, format string
	interval            Interval
	now                 time.Time
	current             string
	compressAfter, keep int
	onError             func(error)
}

func (j cleanJob) run() {
	entries, err := os.ReadDir(j.dir)
	if err != nil {
		j.fail(err)
		return
	}
	cur, ok := j.parse(j.now.Format(j.format)) // 今の間隔の始まり
	if !ok {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == j.current {
			continue
		}
		path := filepath.Join(j.dir, name)
		if base, ok := strings.CutSuffix(name, tmpSuffix); ok {
			// 片付けは 1 つずつしか動かないので、ここで見える途中のファイルは前に止まったときの残り
			if _, _, ok := j.match(base); ok {
				j.fail(remove(path)) // 同じ回の圧縮で使って消えていることもある
			}
			continue
		}
		t, gz, ok := j.match(name)
		if !ok {
			continue
		}
		switch {
		case j.keep > 0 && !t.After(shift(cur, j.interval, -j.keep)):
			j.fail(remove(path))
		case j.compressAfter > 0 && !gz && !t.After(shift(cur, j.interval, -j.compressAfter)):
			j.fail(compress(path))
		}
	}
}

func (j cleanJob) fail(err error) {
	if err != nil && j.onError != nil {
		j.onError(fmt.Errorf("RollingFileWriter: %w", err))
	}
}

// match は name がこのライターのファイル名の形かを確かめ、時刻と圧縮済みかを返す。
func (j cleanJob) match(name string) (t time.Time, gz bool, ok bool) {
	base, gz := strings.CutSuffix(name, ".log"+gzSuffix)
	if !gz {
		if base, ok = strings.CutSuffix(name, ".log"); !ok {
			return time.Time{}, false, false
		}
	}
	if j.prefix != "" {
		if base, ok = strings.CutPrefix(base, j.prefix+"_"); !ok {
			return time.Time{}, false, false
		}
	}
	t, ok = j.parse(base)
	return t, gz, ok
}

// parse は時刻の部分を読む。書き戻して同じ文字列になるものだけを認める（"app_backup" などを外す）。
func (j cleanJob) parse(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(j.format, s, j.now.Location())
	if err != nil || t.Format(j.format) != s {
		return time.Time{}, false
	}
	return t, true
}

// remove は path を消す。もう無ければ何もしない。
func remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// shift は t を n 間隔ずらす。
func shift(t time.Time, i Interval, n int) time.Time {
	switch i {
	case Second:
		return t.Add(time.Duration(n) * time.Second)
	case Minute:
		return t.Add(time.Duration(n) * time.Minute)
	case Hour:
		return t.Add(time.Duration(n) * time.Hour)
	case Day:
		return t.AddDate(0, 0, n)
	case Month:
		return t.AddDate(0, n, 0)
	case Year:
		return t.AddDate(n, 0, 0)
	}
	return t
}

// compress は path を path+".gz" にし、元を消す。途中のファイルを経由するので、
// 止まっても ".gz" が半端に残ることはない。
func compress(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	info, err := src.Stat()
	if err != nil {
		src.Close()
		return err
	}
	tmp := path + tmpSuffix
	dst, err := os.Create(tmp)
	if err != nil {
		src.Close()
		return err
	}
	zw := gzip.NewWriter(dst)
	zw.Name = filepath.Base(path)
	zw.ModTime = info.ModTime()
	_, err = io.Copy(zw, src)
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	src.Close() // Windows では閉じないと消せない
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("compress %s: %w", path, err)
	}
	os.Chtimes(tmp, info.ModTime(), info.ModTime()) // 圧縮したあとも、いつのログか分かるように
	if err := os.Rename(tmp, path+gzSuffix); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Remove(path)
}
