package reliablebc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/relab/goxos/grp"
)

//Function for generating keys
func generateKeys(nodes, keyStrength int) (map[string][]byte, error) {
	keys := make(map[string][]byte)
	for i := 0; i < nodes-1; i++ {
		for j := i + 1; j < nodes; j++ {
			key := make([]byte, keyStrength)
			if _, err := rand.Read(key); err != nil {
				return nil, err
			}
			id := fmt.Sprintf("%d%s%d", i, "-", j)
			keys[id] = key
		}
	}
	return keys, nil
}

//Function for writing the keys to a .json-file
func writeJSON(keys map[string][]byte, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	store := make(map[string]string)
	for k, v := range keys {
		store[k] = hex.EncodeToString(v)
	}
	b, err := json.Marshal(store)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}

//Function for retrieving keys from a .json-file
func readJSON(path string) (map[string][]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, fi.Size())
	n, err := file.Read(buf)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	store := make(map[string]string)
	err = json.Unmarshal(buf[:n], &store)
	if err != nil {
		return nil, err
	}
	keys := make(map[string][]byte)
	for k, v := range store {
		keys[k], err = hex.DecodeString(v)
	}
	if err != nil {
		return nil, err
	}
	return keys, nil
}

//Function for retrieving the keys needed for a single node
func KeysByID(id grp.ID, pathToKeys string) (map[int][]byte, error) {
	allKeys, err := readJSON(pathToKeys)
	keys := make(map[int][]byte)
	for k, v := range allKeys {
		sl := strings.Split(k, "-")
		a, err := strconv.Atoi(sl[0])
		b, err := strconv.Atoi(sl[1])
		if err != nil {
			return nil, err
		}
		if a == id.PxInt() {
			keys[b] = v
		} else if b == id.PxInt() {
			keys[a] = v
		}
	}
	return keys, err
}
