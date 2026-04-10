//go:build !windows

package daemon

import (
	"os"
	"os/signal"
	"syscall"
)

func registerShutdownSignals(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
}

func processExists(process *os.Process) bool {
	return process.Signal(syscall.Signal(0)) == nil
}
