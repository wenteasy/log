package log_test

import (
	"fmt"

	"github.com/wenteasy/log"

	stdLog "log"
	"os"
	"testing"
)

func TestLogger(t *testing.T) {
	logger := log.Get()
	if logger == nil {
		t.Errorf("logger is nil")
	}

	//logger string???
}

func Example() {

	l := stdLog.New(os.Stdout, "Write Test:", stdLog.Lmsgprefix)

	log.Set(l, log.EMERG)

	write(log.DEBUG)
	write(log.INFO)
	write(log.NOTICE)
	write(log.WARN)
	write(log.ERROR)
	write(log.CRIT)
	write(log.ALERT)
	write(log.EMERG)

	// Output:
	// * Now Level Debug
	// Write Test:[Debu]Debug() Write
	// Write Test:[Info]Info() Write
	// Write Test:[Noti]Notice() Write
	// Write Test:[Warn]Warn() Write
	// Write Test:[Erro]Error() Write
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Information
	// Write Test:[Info]Info() Write
	// Write Test:[Noti]Notice() Write
	// Write Test:[Warn]Warn() Write
	// Write Test:[Erro]Error() Write
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Notice
	// Write Test:[Noti]Notice() Write
	// Write Test:[Warn]Warn() Write
	// Write Test:[Erro]Error() Write
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Warning
	// Write Test:[Warn]Warn() Write
	// Write Test:[Erro]Error() Write
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Error
	// Write Test:[Erro]Error() Write
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Critical
	// Write Test:[Crit]Crit() Write
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Alert
	// Write Test:[Aler]Alert() Write
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	// * Now Level Emergency
	// Write Test:[Emer]Emerg() Write
	// Write Test:[Forc]Write() Write
	//
}

func write(lv log.Priority) {

	log.SetLevel(lv)
	logger := log.Get()

	fmt.Println("* Now Level", log.GetLevel())

	logger.Debug("Debug() Write")
	logger.Info("Info() Write")
	logger.Notice("Notice() Write")
	logger.Warn("Warn() Write")
	logger.Error("Error() Write")
	logger.Crit("Crit() Write")
	logger.Alert("Alert() Write")
	logger.Emerg("Emerg() Write")
	logger.Write("Write() Write")
}
