package client

import (
	"fmt"
	"testing"
	"time"
)

var cc *ClientConn

func reqHelper(id string) *Request {
	var seq = uint32(10)
	req := &Request{
		Type: Request_EXEC.Enum(),
		Id:   &id,
		Seq:  &seq,
		Val:  []byte("value"),
	}
	return req
}

func TestClientConn(t *testing.T) {
	done := make(chan bool)
	l := listener("localhost:8100", t)
	defer l.Close()

	go func() {
		c := dial("localhost:8100", t)
		id, _ := generateID() // valid id field
		req := reqHelper(id)
		nwrite(c, req, t)

		req = reqHelper("Fake ID with right length...") // valid id field
		nwrite(c, req, t)

		req = reqHelper("") // invalid id field
		nwrite(c, req, t)
	}()

	go func() {
		c := accept(l, t)
		reqCh := make(chan *Request, 64)
		cc = NewClientConn(c, reqCh)
		go cc.Serve()

		r := <-reqCh
		if r.IsIDInvalid() {
			t.Error("expected valid id field")
		}
		fmt.Println(r)

		r = <-reqCh
		if r.IsIDInvalid() {
			t.Error("expected valid id field")
		}
		fmt.Println(r)

		for cnt := 0; cc.connected; cnt++ {
			time.Sleep(2 * time.Millisecond)
			if cnt > 10 {
				t.Error("expected connection to be closed due to 0 length id field")
			}
		}
		done <- true
	}()
	<-done
	fmt.Println(cc.String())
}

func TestClientConnWrongLenID(t *testing.T) {
	done := make(chan bool)
	l := listener("localhost:8100", t)
	defer l.Close()

	go func() {
		c := dial("localhost:8100", t)
		req := reqHelper("Fake ID with wrong length.") // invalid id field
		nwrite(c, req, t)
	}()

	go func() {
		c := accept(l, t)
		reqCh := make(chan *Request, 64)
		cc = NewClientConn(c, reqCh)
		go cc.Serve()

		for cnt := 0; cc.connected; cnt++ {
			time.Sleep(2 * time.Millisecond)
			if cnt > 10 {
				t.Error("expected connection to be closed due to wrong length id field")
				break
			}
		}
		done <- true
	}()
	<-done
	fmt.Println(cc.String())
}
