package reliablebc

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"strconv"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type AuthManager struct {
	ID   grp.ID
	Keys map[int][]byte
}

func NewAuthManager(id grp.ID, conf *config.Config) *AuthManager {
	pathToKeys := conf.GetString(
		"configKeysPath",
		config.DefConfigKeysPath,
	)
	keys, err := KeysByID(id, pathToKeys)
	if err != nil {
		glog.Errorln("Error in getting keys:", err)
	}
	return &AuthManager{
		ID:   id,
		Keys: keys,
	}
}

func (au *AuthManager) checkMAC(senderID grp.ID, received, value []byte, rnd uint) bool {
	expected := au.generateMAC(senderID, value, rnd)
	return hmac.Equal(received, expected)
}

func (au *AuthManager) generateMAC(destID grp.ID, value []byte, rnd uint) []byte {
	if destID.PxInt() == au.ID.PxInt() {
		return nil
	}
	msg := strconv.AppendInt(value, int64(rnd), 10)
	key := au.Keys[destID.PxInt()]
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)

	return mac.Sum(nil)
}

// ****************************************************************
// Bytestuffing functions
// ****************************************************************

func (au *AuthManager) hashToBytes(hash uint32) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, hash)
	if err != nil {
		glog.V(2).Infoln("binary.Write failed:", err)
	}
	return buf.Bytes()
}
