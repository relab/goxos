// +build integration

package client

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	N       = 3
	cmds    = 500
	runs    = 20
	seqnums = cmds * runs
)

//TODO We should use os.PathSeparator to get proper cross-platform paths here.
var KVS = os.Getenv("GOPATH") + "/src/github.com/relab/goxos/kvs/"
var cfg = KVS + "kvsd/conf/config.ini"

// TODO should we automatically run go install ?

func startServers(t *testing.T) [N]*os.Process {
	kvsd, err := exec.LookPath("kvsd")
	fmt.Println("kvsd:" + kvsd)
	if err != nil {
		t.Fatal("kvsd not in $PATH; kvsd needs to be installed before running this test")
	}
	argv := []string{
		"-id",
		"1",
		"-v=3",
		"-log_dir=" + KVS + "kvsd/log/",
		"-all-cores",
		"-config-file",
		cfg,
	}
	var proc [N]*os.Process

	// start N servers
	for i := 0; i < N; i++ {
		argv[1] = strconv.Itoa(i)
		cmd := exec.Command(kvsd, argv...)
		//cmd.Dir = os.Getenv("GOPATH") + "src\\github.com\\relab\\goxos\\kvs\\kvsd"
		err := cmd.Start()
		if err != nil {
			t.Fatal("Failed to start kvsd:", err)
		}
		proc[i] = cmd.Process
	}
	return proc
}

func startClient(t *testing.T, argv ...string) (*exec.Cmd, io.ReadCloser) {
	kvsc, err := exec.LookPath("kvsc")
	fmt.Println("kvsC:" + kvsc)
	if err != nil {
		t.Fatal("kvsc not in $PATH; kvsc needs to be installed before running this test")
	}
	// start client in the provided mode
	cmd := exec.Command(kvsc, argv...)
	//cmd.Dir = "../kvs/kvsc"

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		t.Fatal("Failed to start kvsc:", err)
	}
	return cmd, stderr
}

func checkClientOutput(out io.ReadCloser, expected []string, t *testing.T) {
	scanner := bufio.NewScanner(out)
	k := 0
	for scanner.Scan() {
		line := scanner.Text()
		if k < len(expected) {
			if strings.Contains(line, expected[k]) {
				k++
			}
		}
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard error:", err)
	}

	if k != len(expected) {
		t.Fatal("Client output did not contain expected output: ", len(expected), ", k: ", k)
	}
}

func TestSyncConnection(t *testing.T) {
	proc := startServers(t)

	// kill servers when done
	defer proc[0].Kill()
	defer proc[1].Kill()
	defer proc[2].Kill()

	// give the servers some time to start
	time.Sleep(1000 * time.Millisecond)

	cmd, stderr := startClient(t,
		"-mode=bench",
		"-runs="+strconv.Itoa(runs),
		"-cmds="+strconv.Itoa(cmds),
		"-noreport",
		"-config-file",
		cfg,
	)

	// text expected in output
	expected := make([]string, seqnums)
	for i := 0; i < seqnums; i++ {
		expected[i] = "seq#: " + strconv.Itoa(i)
	}
	checkClientOutput(stderr, expected, t)

	// wait for client to finish
	if err := cmd.Wait(); err != nil {
		t.Fatal("kvsc failed:", err)
	}
}

func TestSyncReconnect(t *testing.T) {
	proc := startServers(t)

	// kill the remaining servers when done
	defer proc[0].Kill()
	defer proc[1].Kill()

	// give the servers some time to start
	time.Sleep(1000 * time.Millisecond)

	cmd, stderr := startClient(t,
		"-mode=bench",
		"-runs="+strconv.Itoa(runs),
		"-cmds="+strconv.Itoa(cmds),
		"-noreport",
		"-config-file",
		cfg,
	)

	// kill the third process (the leader) after 2 seconds
	time.Sleep(2000 * time.Millisecond)
	proc[2].Kill()

	// text expected in output
	expected := []string{
		"read failed: EOF",
		"connecting to another replica",
		"resending",
		"Done!",
	}
	if runtime.GOOS == "windows" {
		expected[0] = "failed: WSA" //can be read or write
	}
	t.Log("Checking expected output")
	checkClientOutput(stderr, expected, t)

	// Also check that the sequence numbers are in order
	// expected = make([]string, seqnums)
	// for i := 0; i < seqnums; i++ {
	// 	expected[i] = "seq#: " + strconv.Itoa(i)
	// }
	// t.Log("Checking expected seq numbers")
	// checkClientOutput(stderr, expected, t)

	// wait for client to finish
	if err := cmd.Wait(); err != nil {
		t.Fatal("kvsc failed:", err)
	}
}

func TestAsyncConnection(t *testing.T) {
	proc := startServers(t)

	// kill servers when done
	defer proc[0].Kill()
	defer proc[1].Kill()
	defer proc[2].Kill()

	// give the servers some time to start
	time.Sleep(1000 * time.Millisecond)

	cmd, stderr := startClient(t,
		"-mode=bench-async",
		"-runs="+strconv.Itoa(runs),
		"-cmds="+strconv.Itoa(cmds),
		"-noreport",
		"-config-file",
		cfg,
	)

	// text expected in output
	expected := []string{
		"handshake: id accepted",
		"Done!",
	}
	// expected := []string{
	// 	"read failed: EOF",
	// 	"connecting to another replica",
	// 	"resending",
	// 	"Done!",
	// }
	// if runtime.GOOS == "windows" {
	// 	expected[0] = "read failed: WSARecv"
	// }
	checkClientOutput(stderr, expected, t)

	// wait for client to finish
	if err := cmd.Wait(); err != nil {
		t.Fatal("kvsc failed:", err)
	}
}

func TestAsyncReconnect(t *testing.T) {
	proc := startServers(t)

	// kill the remaining servers when done
	defer proc[0].Kill()
	defer proc[1].Kill()

	// give the servers some time to start
	time.Sleep(1000 * time.Millisecond)

	cmd, stderr := startClient(t,
		"-mode=bench-async",
		"-runs="+strconv.Itoa(runs),
		"-cmds="+strconv.Itoa(cmds),
		"-noreport",
		"-config-file",
		cfg,
	)

	// kill the third process (the leader) after 2 seconds
	time.Sleep(2000 * time.Millisecond)
	proc[2].Kill()

	// text expected in output
	expected := []string{
		"handshake: id accepted",
		"Done!",
	}
	// expected := []string{
	// 	"read failed: EOF",
	// 	"connecting to another replica",
	// 	"resending",
	// 	"Done!",
	// }
	// if runtime.GOOS == "windows" {
	// 	expected[0] = "read failed: WSARecv"
	// }
	checkClientOutput(stderr, expected, t)

	// wait for client to finish
	if err := cmd.Wait(); err != nil {
		t.Fatal("kvsc failed:", err)
	}
}
