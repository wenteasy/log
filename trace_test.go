package log_test

import (
	stdLog "log"
	"os"
	"strings"

	"github.com/wenteasy/log"
)

func ExampleLoggerTrace() {

	l := stdLog.New(os.Stdout, "Trace Test:", stdLog.Lmsgprefix)
	log.Set(l, log.DEBUG)
	gg := log.Get()

	defer gg.Trace(gg.TS("TestTrace()"))

	gg.Debug("Debug")

	// Output:
	// Trace Test:========== TestTrace() Start
	// Trace Test:Debug
	// Trace Test:========== TestTrace() End
}

func ExampleSetTrace() {

	l := stdLog.New(os.Stdout, "Trace Config Test:", stdLog.Lmsgprefix)
	log.Set(l, log.INFO)
	gg := log.Get()

	log.SetTrace(log.INFO, strings.Repeat("*", 5), "Function Start", "Function End")

	defer gg.Trace(gg.TS("TestTrace()"))

	gg.Debug("Debug")
	gg.Info("Info")

	// Output:
	// Trace Config Test:***** TestTrace() Function Start
	// Trace Config Test:Info
	// Trace Config Test:***** TestTrace() Function End
}
