package main

import (
	"encoding/gob"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var serveCmd = cli.Command{
	Name:      "serve",
	ShortName: "",
	Usage:     "start serving requests",
	Action:    serve,
}

const socketTestName = "SocketTest"

func socketTestHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	pid := os.Getpid()
	log.Println("Testing socket... returning pid:", pid)
	if err := enc.Encode(pid); err != nil {
		log.Printf("Failed to encode: %v\n", err)
		return err
	}
	return nil
}

func serve(c *cli.Context) {
	l, lerr := net.Listen("unix", socketAddr)
	if lerr != nil {
		var pid int
		// test if there is a server running by sending it a message; it will
		// return its pid if its running; if its not running, we remove the
		// old socket file, which may have been left behind due to a crash.
		err := ipcFunc(socketTestName, func(enc *gob.Encoder, dec *gob.Decoder) error {
			if e := dec.Decode(&pid); e != nil {
				return e
			}
			return nil
		})
		if err != nil {
			log.Println("no server running; removing old socket file.")
			err = os.Remove(socketAddr)
			if err != nil {
				log.Fatalf("failed to remove socket file: %v\n", err)
			}
			// Retrying:
			l, err = net.Listen("unix", socketAddr)
			if err != nil {
				log.Fatalf("retry failed on socket: %s: %v\n", socketAddr, err)
			}
		} else {
			log.Fatalf("%s (pid %d) already running\n", os.Args[0], pid)
		}
	}
	defer l.Close()
	sigHandler(l)
	log.Printf("%s starting to serve... pid: %d\n", os.Args[0], os.Getpid())
	serveLoop(l)
}

func sigHandler(l net.Listener) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		// Wait for a SIGINT or SIGKILL or SIGTERM:
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		// Stop listening (and unlink the socket if unix type):
		l.Close()
		// And we're done:
		os.Exit(0)
	}(sigc)
}

func serveLoop(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Failed connection attempt: %v", err)
			continue
		}
		defer conn.Close()

		dec := gob.NewDecoder(conn)
		enc := gob.NewEncoder(conn)

		var cmd string
		err = dec.Decode(&cmd)
		if err != nil {
			log.Println(err)
			continue
		}
		if handler, ok := cmdHandlers[cmd]; ok {
			err = handler(enc, dec)
			if err != nil {
				if err.Error() == "Shutdown" {
					// exit for loop
					break
				}
				log.Println(err)
				continue
			}
		} else {
			log.Printf("Unknown command: %s\n", cmd)
			continue
		}
	}
	log.Println("Shutdown completed. Goodbye.")
}

// Helper function called by domain socket clients to communicate with launch
// in serve mode. Simply pass the function with the interaction logic to
// ipcFunc() taking as input the encoder and decoder over which interaction
// can take place. See 'list.go' for a request/reply example, 'shutdown.go'
// for a really simple command-only example, and 'submit.go' for a simple one-
// way example.
func ipcFunc(cmd string, f CmdHandler) error {
	conn, err := net.Dial("unix", socketAddr)
	if err != nil {
		// TODO Why not return err here??  I think it is ok to return err.
		// log.Fatalln(err)
		return err
	}
	defer conn.Close()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	if err = enc.Encode(cmd); err != nil {
		return err
	}
	// We allow passing a nil instead of a function, if we only need to encode
	// a simple command and need no reply etc.
	if f != nil {
		return f(enc, dec)
	}
	return nil
}
