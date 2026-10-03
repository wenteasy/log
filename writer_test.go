package log_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wenteasy/log"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRollingFileWriterRotates(t *testing.T) {
	dir := t.TempDir()
	w, err := log.NewRollingFileWriter(dir, log.Day)
	if err != nil {
		t.Fatal(err)
	}
	w.SetPrefix("app")
	now := time.Date(2026, 1, 2, 23, 59, 0, 0, time.Local)
	w.SetNow(func() time.Time { return now })

	w.Write([]byte("first\n"))
	if got := w.Path(); got != filepath.Join(dir, "app_20260102.log") {
		t.Errorf("Path() = %q", got)
	}
	now = now.Add(2 * time.Minute)
	w.Write([]byte("second\n"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	day1 := readFile(t, filepath.Join(dir, "app_20260102.log"))
	day2 := readFile(t, filepath.Join(dir, "app_20260103.log"))
	if !strings.HasPrefix(day1, "first\n") || !strings.HasSuffix(day1, "end here.\n") {
		t.Errorf("day1 = %q", day1)
	}
	if !strings.HasPrefix(day2, "second\n") {
		t.Errorf("day2 = %q", day2)
	}
}

// 既にあるファイルは続きから書き、区切りは行として入る（次の行と繋がらない）。
func TestRollingFileWriterReopen(t *testing.T) {
	dir := t.TempDir()
	for _, line := range []string{"one\n", "two\n"} {
		w, err := log.NewRollingFileWriter(dir, log.None)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(line))
		w.Close()
		w.Close() // 2 回目は何もしない
	}
	got := readFile(t, filepath.Join(dir, "log.log"))
	want := "one\n<-------------- end here.\n\n--------------> since the file exists, starting from here.\ntwo\n<-------------- end here.\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestRollingFileWriterNoMarkers(t *testing.T) {
	dir := t.TempDir()
	w, _ := log.NewRollingFileWriter(filepath.Join(dir, "sub"), log.None) // 無いディレクトリは作る
	w.SetPrefix("x")
	w.DisableMarkers()
	w.Write([]byte("a\n"))
	w.Close()
	w.Write([]byte("b\n")) // 閉じたあとに書けば、また開く
	w.Close()
	if got := readFile(t, filepath.Join(dir, "sub", "x.log")); got != "a\nb\n" {
		t.Errorf("got %q", got)
	}
}
