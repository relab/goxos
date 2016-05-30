// +build !windows

package main

import (
	"os/exec"
	"syscall"
)

// This decouples the provided child process from its parent (this process).
// This prevents the child process from being killed off by the operating
// system, if this process is killed or crashes.
func decoupleParent(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
