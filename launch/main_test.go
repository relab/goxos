// +build launcher

// To run this integration test:
//   go test -tags launcher
package main

import (
	"testing"
	. "time"
)

func TestLaunch(t *testing.T) {
	go serve(nil)
	Sleep(1000 * Millisecond)

	// l := subscribeListener("tcp", "0.0.0.0:9000")
	// go subscribeLoop(l)

	// sshSubscription(
	// 	currentUser().Username,
	// 	filepath.Join(currentUser().HomeDir, ".ssh", "id_rsa"),
	// 	"*",
	// 	l.Addr().String(),
	// 	"localhost:22",
	// )
	// Sleep(1000 * Millisecond)

	loadAndSubmit([]string{"apps"})

	Sleep(10000 * Millisecond)
}
