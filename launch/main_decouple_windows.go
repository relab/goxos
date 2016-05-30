// +build windows

package main

import "os/exec"

// Unsupported on Windows
func decoupleParent(cmd *exec.Cmd) {
}
