package client

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"io"
	"log"
	"net"

	"github.com/relab/goxos/config"
)

//TODO: Make the members entry be a map from hostname:port to net.Conn. This
//way activeConn can be a string with the key for the active member. And if
//activeConn="all", then it means all members have a connection.

//TODO: Let the initial connection be to any of the servers. The handshake
//reply should contain the full list of members.

//TODO: Make a separate Conn struct for handling seq# and responseMap,
//writeLock etc. Do we need a separate seq# for each Conn (when using multiple
//connections?)

// type connection struct {
// 	conn      net.Conn
// 	seq       uint32
// 	respMap   responseMap
// 	writeLock *sync.Mutex
// }

type connManager struct {
	membID     []byte
	members    map[string]net.Conn
	activeConn string
	conf       *config.Config
}

func newConnManager(members []string, config *config.Config) *connManager {
	conn := &connManager{
		membID:  MembID(members),
		members: make(map[string]net.Conn),
		conf:    config,
	}
	for _, member := range members {
		conn.members[member] = nil
	}
	return conn
}

// MembId returns the membership identifier for the given membership. The
// membership identifier is expected to be identical for the same membership.
// Thus, it does not distinguish between two versions of the same membership
// that may have existed at different times during an execution.
func MembID(members []string) []byte {
	h := sha1.New()
	for _, member := range members {
		io.WriteString(h, member)
	}
	return h.Sum(nil)
}

// MembIdEqual returns true if aId and bId are equal.
func MembIDEqual(aID, bID []byte) bool {
	return bytes.Equal(aID, bID)
}

// Members returns the set of members managed by the connection manager.
func (cm *connManager) Members() []string {
	var membs []string
	for member := range cm.members {
		membs = append(membs, member)
	}
	return membs
}

func (cm *connManager) Size() int {
	return len(cm.members)
}

func (cm *connManager) Close() {
	for _, member := range cm.members {
		if member != nil {
			member.Close()
		}
	}
}

func (cm *connManager) update(members []string) {
	newMembers := make(map[string]net.Conn)
	for _, member := range members {
		newMembers[member] = nil
	}
	foundActive := false
	if _, present := newMembers[cm.activeConn]; present {
		// Member with active connection is also in the new membership.
		// Copy connection object to new data structure.
		newMembers[cm.activeConn] = cm.members[cm.activeConn]
		foundActive = true
	}
	cm.members = newMembers
	cm.membID = MembID(members)
	if !foundActive {
		// Connect to a new member, since no active member found.
		cm.connect()
	}
}

func (cm *connManager) connect() error {
	for i := 0; i < cm.conf.GetInt("cycleListMax", config.DefCycleListMax); i++ {
		for member := range cm.members {
			log.Printf("connect: connecting to %v", member)
			conn, err := net.DialTimeout("tcp", member, cm.conf.GetDuration("dialTimeout", config.DefDialTimeout))
			if err == nil {
				log.Printf("connect: succeeded connecting to %v", member)
				cm.members[member] = conn
				cm.activeConn = member
				return nil
			}
			log.Printf("connect: %v", err)
			continue
		}
	}
	return errors.New("failed connecting to any member nodes")
}
