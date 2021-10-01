package log

import (
	"os"
	"path/filepath"
	"time"

	"golang.org/x/xerrors"
)

type RollingFileWriter struct {
	prefix string
	path   string

	format string

	suffix string
	target *os.File
}

type Interval int

const (
	Second Interval = iota
	Minite
	Hour
	Day
	Month
	Year
	None
)

// 固定フォーマットの取得
// SetFormat()により、設定可能
func (i Interval) getFormat() string {
	f := ""
	switch i {
	case Second:
		f = "20060102150405"
	case Minite:
		f = "200601021504"
	case Hour:
		f = "2006010215"
	case Day:
		f = "20060102"
	case Month:
		f = "200601"
	case Year:
		f = "2006"
	}
	return f
}

// Writerの新規作成
func NewRollingFileWriter(path string, i Interval) (*RollingFileWriter, error) {

	w := RollingFileWriter{}

	//指定した間隔のフォーマットを取得
	w.format = i.getFormat()
	w.path = path

	w.prefix = ""
	w.suffix = ""

	return &w, nil
}

// フォーマットの指定
// None以外の時に指定した場合でも、フォーマットにより新規作成する為、
// 間違ったフォーマットを指定した場合にその間隔にはならない
func (w *RollingFileWriter) SetFormat(f string) {
	w.format = f
}

// 書き込み
func (w *RollingFileWriter) Write(p []byte) (int, error) {

	f := time.Now().Format(w.format)

	if f != w.suffix {
		// 同期をとる
		w.suffix = f
		err := w.setTarget()
		if err != nil {
			return -1, xerrors.Errorf("setTarget error : %w", err)
		}
	}

	return w.target.Write(p)
}

const (
	logHeader = "\n--------------> "
	logFooter = "\n<-------------- "
	startLog  = logFooter + "end here."
	endLog    = logHeader + "since the file exists,starting from here."
)

// 閉じる
func (w *RollingFileWriter) Close() error {
	if w.target != nil {
		w.target.Write([]byte(endLog))
		return w.target.Close()
	}
	return nil
}

func (w *RollingFileWriter) getFileName() string {
	f := w.suffix + ".log"
	if w.prefix != "" {
		f = w.prefix + "_" + f
	}
	return f
}

// ターゲット
func (w *RollingFileWriter) setTarget() error {

	w.Close()
	path := filepath.Join(w.path, w.getFileName())

	var err error
	_, err = os.Stat(path)

	if err == nil {
		// 存在した場合
		w.target, err = os.Open(path)
		if err != nil {
			return xerrors.Errorf("open error: %w", err)
		}

		//すでに存在した為、追加することを書き込む
		w.target.Write([]byte(startLog))

	} else {
		w.target, err = os.Create(path)
		if err != nil {
			return xerrors.Errorf("create error: %w", err)
		}
	}
	return nil
}
