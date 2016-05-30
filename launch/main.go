// The launch command is used to start services. To be able to start services
// on the local machine, there must be an instance of launch that runs in
// serve mode on the local machine. A service may be started in several ways,
// either using a configuration file specifying the service(s) to start or a
// service can be submitted to launch for immediate execution. To start a
// service from a remote machine, we can simply call the launch command using
// ssh or mosh or some other mechanism that allows command execution from a
// remote machine. The command line interface is partially inspired by the
// launchctl command on OS X; see 'man launchctl' for more details (I may add
// more command line options later to allow improved compatability with
// launchctl, although that is not a goal in itself.)
package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const (
	// Should ideally be in /var/run on Unix, but that requires root access
	socketAddr = "/tmp/launch.sock"
	// the file type used for configuration files.
	fileType = "json"
)

//TODO: store stuff on disk?
//TODO: Store things in a data store (kvs)

// CmdHandler is the handler that a command must provide to the serve mode
// for it to be able to process the request and return a reply if needed.
// See 'submitHandler' and 'listHandler' for examples.
type CmdHandler func(*gob.Encoder, *gob.Decoder) error

// To be able to serve new commands, a corresponding command handler must
// be added to the map below. See 'submit.go' and 'list.go' for examples.
var cmdHandlers = map[string]CmdHandler{
	submitName:     submitHandler,
	listName:       listHandler,
	stopName:       stopHandler,
	shutdownName:   shutdownHandler,
	subscribeName:  subscribeHandler,
	socketTestName: socketTestHandler,
}

// TODO Make part of this command line structure into a library that is easier to test and use

func main() {
	app := cli.NewApp()
	app.Name = os.Args[0]
	app.Usage = "Launch programs on remote nodes."
	app.Version = "0.0.1"
	app.Author = "Hein Meling"
	app.Email = "hein.meling@uis.no"
	app.Commands = []cli.Command{
		serveCmd,
		submitCmd,
		loadCmd,
		saveCmd,
		listCmd,
		stopCmd,
		shutdownCmd,
		subscribeCmd,
	}
	app.Flags = flags
	app.Run(os.Args)
}

//TODO Send special struct or interface over channel (should contain ProcessState and Process info)
type Event struct {
	Label string
	Path  string
	Pid   int
	Succ  bool
}

func (e Event) String() string {
	return fmt.Sprintf("%s: %s (%d): %t", e.Label, e.Path, e.Pid, e.Succ)
}

func launch(srv *ServiceInfo) {
	if running.contains(srv.Label) {
		log.Printf("%v already running\n", srv.Label)
		return
	}
	cmd := exec.Command(srv.Program, srv.ProgramArgs...)
	decoupleParent(cmd)
	err := cmd.Start()
	if err != nil {
		log.Printf("%v failed to start; %v\n", cmd.Path, err)
		return
	}
	log.Printf("Started: %s from %s\n", srv.Label, cmd.Path)
	running.add(srv.Label, srv, cmd)
	defer running.remove(srv.Label)

	if err = cmd.Wait(); err != nil {
		log.Printf("%v returned with error: %v\n", cmd.Path, err)
	} else {
		log.Printf("%v returned without error\n", cmd.Path)
	}
	ev := &Event{srv.Label, cmd.Path, cmd.Process.Pid, cmd.ProcessState.Success()}
	notify(ev)
}

// Who owns the lockfile?
// func (l Lockfile) GetOwner() (*os.Process, error) {
// 	name := string(l)

// 	// Ok, see, if we have a stale lockfile here
// 	content, err := ioutil.ReadFile(name)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var pid int
// 	_, err = fmt.Sscanln(string(content), &pid)
// 	if err != nil {
// 		return nil, ErrInvalidPid
// 	}

// 	// try hard for pids. If no pid, the lockfile is junk anyway and we delete it.
// 	if pid > 0 {
// 		p, err := os.FindProcess(pid)
// 		if err != nil {
// 			return nil, err
// 		}
// 		err = p.Signal(os.Signal(syscall.Signal(0)))
// 		if err == nil {
// 			return p, nil
// 		}
// 		errno, ok := err.(syscall.Errno)
// 		if !ok {
// 			return nil, err
// 		}

// 		switch errno {
// 		case syscall.ESRCH:
// 			return nil, ErrDeadOwner
// 		case syscall.EPERM:
// 			return p, nil
// 		default:
// 			return nil, err
// 		}
// 	} else {
// 		return nil, ErrInvalidPid
// 	}
// 	panic("Not reached")
// }

func notify(ev *Event) {
	for k, sub := range subscribers {
		if m, _ := path.Match(sub.filter, ev.Label); m {
			fmt.Println("notifying:", k)
			// We assume that the filter provided with the subscription
			// does not fail; this is checked in subscriberHandler()
			sub.failCh <- ev
		}
	}
}
