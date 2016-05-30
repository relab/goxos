package client

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"time"
)

const randbits = 64

//TODO define the semantics of the returns for this method. Currently it
//returns empty string and err, but it should probably return nil, err.

func generateID() (id string, err error) {
	b := new(bytes.Buffer)

	// hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	b.Write([]byte(hostname)) //Lookup IP instead?

	// time
	now := time.Now()
	nowAsGob, err := now.GobEncode()
	if err != nil {
		return "", err
	}
	b.Write(nowAsGob)

	// random bits
	rb := make([]byte, randbits/8)
	n, err := io.ReadFull(rand.Reader, rb)
	if err != nil {
		return "", err
	}
	if n != len(rb) {
		return "", errors.New("random bits length mismatch")
	}
	b.Write(rb)

	hasher := sha1.New()
	hasher.Write(b.Bytes())
	id = base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return id, nil
}
