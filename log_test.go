package log_test

import (
	"fmt"

	own "github.com/wenteasy/log"

	"log"
	"os"
	"testing"
)

func TestLogger(t *testing.T) {
	logger := own.Get()
	if logger == nil {
		t.Errorf("logger is nil")
	}

	l := own.GetLogger()
	if l != nil {
		t.Errorf("internal logger is not nil")
	}

	lv := own.GetLevel()
	if lv != own.EMERG {
		t.Errorf("default logger level is not emergency")
	}
}

func TestWrite(t *testing.T) {

	l := log.New(os.Stdout, "Write Test:", log.Ldate|log.Ltime|log.Lshortfile)

	own.SetLogger(l)

	write(own.DEBUG)
	write(own.INFO)
	write(own.NOTICE)
	write(own.WARN)
	write(own.ERROR)
	write(own.CRIT)
	write(own.ALERT)
	write(own.EMERG)
}

func write(lv own.Priority) {

	own.SetLevel(lv)

	fmt.Println("* Now Level", own.GetLevel())

	logger := own.Get()
	logger.Debug("Write() %v", own.DEBUG)
	logger.Info("Write() %v", own.INFO)
	logger.Notice("Write() %v", own.NOTICE)
	logger.Warn("Write() %v", own.WARN)
	logger.Error("Write() %v", own.ERROR)
	logger.Crit("Write() %v", own.CRIT)
	logger.Alert("Write() %v", own.ALERT)
	logger.Emerg("Write() %v", own.EMERG)
}
