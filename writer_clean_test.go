package log_test

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/wenteasy/log"
)

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func readGzip(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// 古さはファイル名の日付で決まる。CompressAfter(3) で 3 日前から圧縮、Keep(7) で 7 日前から消す。
// 名前の形が合わないものには触らない。
func TestRollingFileWriterCompressAndKeep(t *testing.T) {
	dir := t.TempDir()
	for d := 1; d <= 9; d++ {
		name := filepath.Join(dir, "app_2026010"+string(rune('0'+d))+".log")
		os.WriteFile(name, []byte(filepath.Base(name)+"\n"), 0o644)
	}
	others := []string{
		"app_backup.log", "other_20260101.log", "app_20260101.txt", "app_2026011.log", "20260101.log",
	}
	for _, n := range others {
		os.WriteFile(filepath.Join(dir, n), nil, 0o644)
	}
	os.WriteFile(filepath.Join(dir, "app_20251231.log.gz"), nil, 0o644)     // 圧縮済みでも古ければ消す
	os.WriteFile(filepath.Join(dir, "app_20260104.log.gz.tmp"), nil, 0o644) // 前に止まった圧縮の残り

	w, err := log.NewRollingFileWriter(dir, log.Day)
	if err != nil {
		t.Fatal(err)
	}
	w.SetPrefix("app")
	w.DisableMarkers()
	w.SetCompressAfter(3)
	w.SetKeep(7)
	w.SetErrorHandler(func(err error) { t.Errorf("clean: %v", err) })
	w.SetNow(func() time.Time { return time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local) })
	w.Write([]byte("today\n"))
	w.WaitClean()
	w.Close()

	want := []string{
		// 7 日前（20260103）より前は消える
		"app_20260104.log.gz", "app_20260105.log.gz", "app_20260106.log.gz", "app_20260107.log.gz", // 3〜6 日前
		"app_20260108.log", "app_20260109.log", "app_20260110.log", // 0〜2 日前
	}
	want = append(want, others...)
	slices.Sort(want)
	if got := listDir(t, dir); !slices.Equal(got, want) {
		t.Errorf("files\ngot  %q\nwant %q", got, want)
	}
	if got := readGzip(t, filepath.Join(dir, "app_20260105.log.gz")); got != "app_20260105.log\n" {
		t.Errorf("app_20260105.log.gz = %q", got)
	}
}

// 切り替えたときにも片付ける。CompressAfter(1) なら前のファイルがすぐ圧縮される。
func TestRollingFileWriterCompressOnSwitch(t *testing.T) {
	dir := t.TempDir()
	w, _ := log.NewRollingFileWriter(dir, log.Day)
	w.DisableMarkers()
	w.SetCompressAfter(1)
	now := time.Date(2026, 1, 2, 23, 59, 0, 0, time.Local)
	w.SetNow(func() time.Time { return now })

	w.Write([]byte("first\n"))
	w.WaitClean()
	now = now.Add(2 * time.Minute)
	w.Write([]byte("second\n"))
	w.WaitClean()
	w.Close()

	if got := listDir(t, dir); !slices.Equal(got, []string{"20260102.log.gz", "20260103.log"}) {
		t.Errorf("files = %q", got)
	}
	if got := readGzip(t, filepath.Join(dir, "20260102.log.gz")); got != "first\n" {
		t.Errorf("20260102.log.gz = %q", got)
	}
}

// 片付けのエラーは SetErrorHandler に渡る。その中から w へ書いてもよい（ロックを持たずに呼ぶ）。
func TestRollingFileWriterCleanError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "20260101.log"), nil, 0o644)
	os.Mkdir(filepath.Join(dir, "20260101.log.gz.tmp"), 0o755) // 途中のファイルを作れなくする

	w, _ := log.NewRollingFileWriter(dir, log.Day)
	w.DisableMarkers()
	w.SetCompressAfter(1)
	w.SetNow(func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.Local) })
	var mu sync.Mutex
	var errs []error
	w.SetErrorHandler(func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
		w.Write([]byte(err.Error() + "\n"))
	})
	w.Write([]byte("x\n"))
	w.WaitClean()
	w.Close()

	if len(errs) != 1 {
		t.Errorf("errors = %v", errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "20260101.log")); err != nil {
		t.Errorf("the source should be left when compressing fails: %v", err)
	}
}

// 決めていなければ何もしない（既定は消さない・圧縮しない）。
func TestRollingFileWriterNoCleanByDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "20200101.log"), nil, 0o644)
	w, _ := log.NewRollingFileWriter(dir, log.Day)
	w.Write([]byte("x\n"))
	w.WaitClean()
	w.Close()
	if _, err := os.Stat(filepath.Join(dir, "20200101.log")); err != nil {
		t.Error(err)
	}
}
