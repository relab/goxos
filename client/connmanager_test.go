package client

import (
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/relab/goxos/config"
)

var cm *connManager
var listeners []net.Listener

var initialMembership = []string{
	"localhost:8090",
	"localhost:8092",
	"localhost:8094",
}

func TestClientConnManager(t *testing.T) {
	cm = newConnManager(initialMembership, &config.Config{})
}

func TestMembId(t *testing.T) {
	newMembership := []string{
		"localhost:8090",
		"localhost:8092",
		"localhost:8096",
	}
	if !MembIDEqual(MembID(initialMembership), MembID(initialMembership)) {
		t.Error("The two memberships should be equal")
	}
	if MembIDEqual(MembID(initialMembership), MembID(newMembership)) {
		t.Error("The two memberships should not be equal")
	}
	if !MembIDEqual(cm.membID, MembID(initialMembership)) {
		t.Error("The two memberships should be equal")
	}
	if MembIDEqual(cm.membID, MembID(newMembership)) {
		t.Error("The two memberships should not be equal")
	}
}

func TestConnect(t *testing.T) {
	err := cm.connect()
	if err == nil {
		t.Error("We don't have any listener, so we expect err != nil")
	}
	expErr := "failed connecting to any member nodes"
	if err.Error() != expErr {
		t.Errorf("expected: '%v', got: '%v'", expErr, err)
	}

	// Setup listener and try to connect
	listeners = make([]net.Listener, cm.Size())
	for i, member := range cm.Members() {
		listeners[i] = listener(member, t)
		err = cm.connect()
		if err != nil {
			t.Errorf("could not connect, got '%v'", err)
		}
		if cm.activeConn != member {
			t.Errorf("active connection %v, expected %v", cm.activeConn, member)
		}
		listeners[i].Close()
	}
}

func setupListeners(t *testing.T) {
	listeners = make([]net.Listener, cm.Size())
	for i, member := range cm.Members() {
		listeners[i] = listener(member, t)
	}
}

func updateListeners(newMembership []string, t *testing.T) {
	memberSet := make(map[string]bool)
	for _, member := range cm.Members() {
		memberSet[member] = true
	}
	for _, member := range newMembership {
		if memberSet[member] {
			continue
		} else {
			fmt.Printf("New listener: %v\n", member)
			newListener := listener(member, t)
			listeners = append(listeners, newListener)
		}
	}
}

func closeListeners() {
	for _, listener := range listeners {
		listener.Close()
	}
}

func equal(a, b []string) bool {
	if len(a) == len(b) {
		mSet := make(map[string]bool)
		for _, member := range a {
			mSet[member] = true
		}
		for _, member := range b {
			if !mSet[member] {
				return false
			}
		}
	} else {
		return false
	}
	return true
}

func TestUpdate(t *testing.T) {
	setupListeners(t)
	defer closeListeners()
	cm.connect()
	if !equal(cm.Members(), initialMembership) {
		t.Errorf("membership not installed (%v)", cm.members)
	}
	if cm.activeConn == "" {
		t.Errorf("active connection %v, expected non-empty string", cm.activeConn)
	}

	newMembership := []string{
		"localhost:8090",
		"localhost:8092",
		"localhost:8096",
	}
	updateListeners(newMembership, t)
	cm.update(newMembership)
	checkConnections(newMembership, t)

	newMembership = []string{
		"localhost:8098",
		"localhost:8092",
		"localhost:8096",
	}
	updateListeners(newMembership, t)
	cm.update(newMembership)
	checkConnections(newMembership, t)

	newMembership = []string{
		"localhost:8050",
		"localhost:8052",
		"localhost:8054",
	}
	updateListeners(newMembership, t)
	cm.update(newMembership)
	checkConnections(newMembership, t)

	newMembership = []string{
		"localhost:8050",
		"localhost:8052",
		"localhost:8054",
		"localhost:8056",
		"localhost:8058",
		"localhost:8060",
	}
	updateListeners(newMembership, t)
	cm.update(newMembership)
	checkConnections(newMembership, t)

	cm.Close()
}

func checkConnections(newMembership []string, t *testing.T) {
	if !equal(cm.Members(), newMembership) {
		t.Errorf("membership not updated\n(%v)\n(%v)\n", cm.Members(), newMembership)
	}
	if cm.activeConn == "" {
		t.Errorf("active connection %v, expected non-empty string", cm.activeConn)
	}
	for member, conn := range cm.members {
		if cm.activeConn == member {
			if conn == nil {
				t.Errorf("expected non-nil connection for %v", member)
			}
		} else {
			if conn != nil {
				t.Errorf("expected nil connection for %v, got: %v", member, conn.RemoteAddr())
			}
		}
		if conn == nil {
			log.Printf("conn[%v]: nil", member)
		} else {
			log.Printf("conn[%v]: %v", member, conn.RemoteAddr())
		}
	}
}
