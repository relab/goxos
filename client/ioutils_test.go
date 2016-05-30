package client

import (
	"net"
	"strconv"
	"testing"
	"time"
)

const iter = 100

var done chan bool

// helper function to set up listener on the given address; remember to close
// the connection at the end of each test case.
func listener(adr string, t *testing.T) net.Listener {
	l, err := net.Listen("tcp", adr)
	if err != nil {
		t.Fatal("Did you forget to close a connection?", err)
	}
	return l
}

func dial(adr string, t *testing.T) net.Conn {
	c, err := net.Dial("tcp", adr)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func accept(l net.Listener, t *testing.T) net.Conn {
	c, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func nwrite(c net.Conn, req *Request, t *testing.T) {
	err := write(c, req)
	if err != nil {
		t.Fatal(err)
	}
}

func nread(c net.Conn, resp *Response, t *testing.T) {
	err := read(c, resp)
	if err != nil {
		t.Fatal(err)
	}
}

func doWrite(c net.Conn, i int, t *testing.T) {
	myid := strconv.Itoa(i)
	hello := &Request{Type: Request_HELLO.Enum(), Id: &myid}
	if err := write(c, hello); err != nil {
		t.Errorf("write failed: %v", err)
	}
}

// TODO: This test can fail occationally. Figure out why. It may be totally
// reasonable, but it may also not be.
//
// --- FAIL: TestWriteRead (0.07 seconds)
// 	ioutils_test.go:72: expected 0, got: 1
// 	ioutils_test.go:72: expected 2, got: 0
// 	ioutils_test.go:72: expected 1, got: 2

func doRead(l net.Listener, t *testing.T) {
	c := accept(l, t)
	prevID := -1
	for {
		var resp Response
		if err := read(c, &resp); err != nil {
			t.Errorf("read failed: %v", err)
		}
		id, _ := strconv.Atoi(resp.GetId())
		if id != prevID+1 {
			t.Errorf("expected %d, got: %d", prevID+1, id)
		}
		prevID = id
		if id == iter-1 {
			done <- true
		}
	}
}

func TestWriteRead(t *testing.T) {
	done = make(chan bool)
	l := listener(":8100", t)
	defer l.Close()
	go doRead(l, t)

	d := dial("localhost:8100", t)
	for i := 0; i < iter; i++ {
		go doWrite(d, i, t)
		// delay writing to allow interleaving with reads
		time.Sleep(5 * time.Microsecond)
	}
	<-done
}
