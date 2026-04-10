//go:build windows

package daemon

import (
	"os"
	"os/signal"
	"syscall"
)

func registerShutdownSignals(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, os.Interrupt)
}

// processExists checks if the process is still alive on Windows.
// process.Signal(0) is not supported on Windows, so we use OpenProcess instead.
func processExists(process *os.Process) bool {
	const synchronize = 0x00100000
	h, err := syscall.OpenProcess(synchronize, false, uint32(process.Pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}
