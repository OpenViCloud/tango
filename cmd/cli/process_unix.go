//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func stopProcess(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}
