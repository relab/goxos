package main

import (
	"encoding/gob"
	"io"
	"log"
	"net"
	"os"
	"path"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const subscribeName = "subscribe"

var subscribeCmd = cli.Command{
	Name:      subscribeName,
	ShortName: "sub",
	Usage:     "start a subscriber",
	Action:    subscribe,
	Flags:     subscribeFlags,
}

var subscribeFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "net, n",
		Value: "tcp",
		Usage: "Network (unix/tcp) to use for receiving notifications",
	},
	cli.StringFlag{
		Name:  "addr, a",
		Value: "0.0.0.0:9000",
		Usage: "Host:Port on which to receive notifications",
	},
	cli.StringFlag{
		Name:  "filter, f",
		Value: "*",
		Usage: "Match labels of interest; default is all labels",
	},
	cli.BoolFlag{
		Name:  "serve, s",
		Usage: "If set, start receiver for notifications",
	},
}

func subscribe(c *cli.Context) {
	notifyAddr := c.String("addr")
	if c.Bool("serve") {
		netw := c.String("net")
		l := subscribeListener(netw, notifyAddr)
		go subscribeLoop(l)

		// TODO SSH to the node of interest (all nodes?)
		sshSubscription(
			c.GlobalString("user"),
			c.GlobalString("keypath"),
			c.String("filter"),
			l.Addr().String(),
			"localhost:22",
		)
		select {}
	} else {
		log.Println("Sending subscribe to", notifyAddr)
		sendSubscribe(notifyAddr, c.String("filter"))
	}
}

func subscribeListener(netw, addr string) net.Listener {
	l, err := net.Listen(netw, addr)
	if err != nil {
		log.Fatalf("subscribe: %v\n", err)
	}
	defer l.Close()
	sigHandler(l)
	log.Printf("Listening for notifications on %s\n", l.Addr().String())
	return l
}

// send (with ssh) the provided subscription filter to host. The rcvAddr is
// the endpoint on which notifications are expected.
func sshSubscription(userName, privKeyPath, filter, rcvAddr, host string) {
	log.Printf("%s %s\n", userName, privKeyPath)
	config, err := sshAuthSock(userName)
	if err != nil {
		log.Println("Could not authenticate with ssh-agent:", err)
		log.Println("Trying with password-less public-key.")
		config, err = sshConfig(userName, privKeyPath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	cmd := os.Args[0] + " " + subscribeName + " -a '" + rcvAddr + "' -f '" + filter + "'"
	err = sshRun(config, host, cmd)
	if err != nil {
		log.Fatalln("sshRun: failed to execute command:", err)
	}
	log.Printf("%s %s\n", userName, privKeyPath)
}

// This needs to be invoked from each local machine to be able to connect over
// domain sockets to the local 'launch' in serve mode. Thus, one needs to use
// ssh to insert the relevant subscription.
func sendSubscribe(notifyAddr, filter string) {
	err := ipcFunc(subscribeName, func(enc *gob.Encoder, dec *gob.Decoder) error {
		if e := enc.Encode(notifyAddr); e != nil {
			return e
		}
		return enc.Encode(filter)
	})
	if err != nil {
		log.Println(err)
	}
}

type eventSubscriber struct {
	failCh chan *Event
	filter string
}

var (
	subscribers = make(map[string]*eventSubscriber)
)

func subscribeHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	var callbackAddr, filter string
	err := dec.Decode(&callbackAddr)
	if err != nil {
		return err
	}
	err = dec.Decode(&filter)
	if err != nil {
		return err
	}
	log.Printf("callbackAddr: %s", callbackAddr)
	log.Printf("filter: %s", filter)
	_, err = path.Match(filter, "")
	if err != nil {
		// Filter is bad
		return err
	}
	sub := &eventSubscriber{failCh: make(chan *Event, 10), filter: filter}
	subscribers[callbackAddr] = sub

	go func(callbackAddr string) {
		conn, err := net.Dial("tcp", callbackAddr)
		if err != nil {
			log.Println("Failed to connect to subscriber", err)
		}
		defer conn.Close()
		encoder := gob.NewEncoder(conn)
		failChan := subscribers[callbackAddr].failCh

		for {
			notification := <-failChan
			log.Println("notifying failure of:", notification)

			err = encoder.Encode(notification)
			if err != nil {
				log.Println("Failed to notify subscriber", err)
				break
			}
		}
	}(callbackAddr)

	return nil
}

func subscribeLoop(l net.Listener) {
	var notification Event
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Failed connection attempt:", err)
			continue
		}
		defer conn.Close()
		log.Println("Connection from:", conn.RemoteAddr())

		dec := gob.NewDecoder(conn)
		for {
			err = dec.Decode(&notification)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("Failed to decode notification:", err)
				continue
			}
			log.Printf("Service %v failed.", notification)
		}
	}
}
