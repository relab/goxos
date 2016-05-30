package main

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

type ServiceInfo struct {
	Label       string   `plist:"Label"`
	Program     string   `plist:"Program"`
	ProgramArgs []string `plist:"ProgramArguments,omitempty"`
	Disabled    bool     `plist:"Disabled,omitempty"`
	// TODO: ssh host/public/private key? (do we need user name?) No, these
	// are independent of the service info? However, in some cases it may make
	// sense to run a program as a specific user?
}

func (s *ServiceInfo) String() string {
	return fmt.Sprintf("label: %s, prg: %s, args: %v, run it: %v",
		s.Label, s.Program, s.ProgramArgs, !s.Disabled)
}

type Entry struct {
	*ServiceInfo
	*exec.Cmd
}

type Services struct {
	s map[string]*Entry
	*sync.Mutex
}

func (r *Services) contains(label string) bool {
	r.Lock()
	defer r.Unlock()
	_, exists := running.s[label]
	return exists
}

func (r *Services) add(label string, srv *ServiceInfo, cmd *exec.Cmd) {
	r.Lock()
	defer r.Unlock()
	r.s[label] = &Entry{srv, cmd}
}

func (r *Services) remove(label string) {
	r.Lock()
	defer r.Unlock()
	delete(r.s, label)
}

func (r *Services) entry(label string) *Entry {
	r.Lock()
	defer r.Unlock()
	return r.s[label]
}

func (r *Services) killall() {
	r.Lock()
	defer r.Unlock()
	for label, cmd := range r.s {
		err := cmd.Process.Kill()
		if err != nil {
			log.Printf("%s could not be killed: %v\n", label, err)
		}
	}
}

func (r *Services) services() []string {
	r.Lock()
	defer r.Unlock()
	var running []string
	for label := range r.s {
		running = append(running, label)
	}
	return running
}

var (
	// set of running services: label -> *Entry
	running = &Services{make(map[string]*Entry), new(sync.Mutex)}
)
