package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/gob"
	"io/ioutil"
	"sort"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const hashFilename = "statehash"

func (gh *GoxosHandler) GetState(slotMarker uint) (sm uint, state []byte) {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	encoder.Encode(gh.kvmap)
	return slotMarker, buf.Bytes()
}

func (gh *GoxosHandler) SetState(state []byte) error {
	buf := bytes.NewBuffer(state)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(&gh.kvmap); err != nil {
		return err
	}
	return nil
}

func (gh *GoxosHandler) loadState(filePath string) error {
	glog.V(1).Infoln("loading state from", filePath)

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	glog.V(1).Infof("state has size of %d bytes, setting state", len(file))
	if err = gh.SetState(file); err != nil {
		return err
	}

	glog.V(1).Info("loading of initial state was successful")

	return nil
}

func (gh *GoxosHandler) writeStateHash() error {
	// Map key iteration order is randomized. The bytes from gob encoding
	// the map is therefore not deterministic and results in different
	// hashes.
	glog.V(1).Info("generating state hash...")

	// Get all keys
	keys := make([]string, len(gh.kvmap))
	i := 0
	for k := range gh.kvmap {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	// Write every key and value to hasher in key sorted order
	hasher := sha1.New()
	for i := range keys {
		hasher.Write([]byte(keys[i]))
		hasher.Write(gh.kvmap[keys[i]])
	}
	stateFingerprint := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	// Debugging information
	glog.V(1).Infof("hash generated using %d map entries", len(keys))
	if len(gh.kvmap) > 1 {
		glog.V(1).Infof("first entry: %s - %s",
			keys[0], gh.kvmap[keys[0]])
		glog.V(1).Infof("last entry: %s - %s",
			keys[len(gh.kvmap)-1], gh.kvmap[keys[len(gh.kvmap)-1]])
	}
	glog.V(1).Infoln("resulting hash:", stateFingerprint)

	return ioutil.WriteFile(hashFilename, []byte(stateFingerprint), 0644)
}
