package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/kvs/bgen"
	kc "github.com/relab/goxos/kvs/common"
)

func runExp() {
	elog.Enable()
	defer elog.Flush()
	if *keyset != "" {
		runExpWithDetKeys()
	} else {
		runExpWithRandKeys()
	}
}

func runExpWithDetKeys() {
	keySet, err := loadKeySet()
	if err != nil {
		log.Fatalln("error loading keyset:", err)
	}
	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup
	for i := 0; i < *nclients; i++ {
		wg.Add(1)
		go func() {
			serviceConnection, err := client.Dial(clientConfig)
			if err != nil {
				log.Fatalln("Error dailing cluster:", err)
			}

			if *prewait > 0 {
				time.Sleep(*prewait)
			}

			var (
				reqsent time.Time
				kvreq   kc.MapRequest
				value   = make([]byte, *vl)
			)

			for i := 0; i < *cmds; i++ {
				bgen.GetBytes(value)
				kvreq = kc.MapRequest{Ct: kc.Write,
					Key:   keySet[rand.Intn(len(keySet))],
					Value: value,
				}
				kvreq.Marshal(buffer)
				reqsent = time.Now()
				response := <-serviceConnection.Send(buffer.Bytes())
				if response.Err != nil {
					log.Fatalln("Response error:", response.Err)
				}
				elog.Log(e.NewTimedEvent(e.ClientRequestLatency, reqsent))
				buffer.Reset()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func loadKeySet() (keySet [][]byte, err error) {
	file, err := ioutil.ReadFile(*keyset)
	if err != nil {
		return nil, err
	}

	b := new(bytes.Buffer)
	if _, err := b.Write(file); err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(b)
	if err := decoder.Decode(&keySet); err != nil {
		return nil, err
	}

	return keySet, nil
}

func runExpWithRandKeys() {
	var wg sync.WaitGroup
	for i := 0; i < *nclients; i++ {
		wg.Add(1)
		go func() {
			serviceConnection, err := client.Dial(clientConfig)
			if err != nil {
				log.Fatalln("Error dailing cluster:", err)
			}

			if *prewait > 0 {
				time.Sleep(*prewait)
			}

			var (
				reqsent time.Time
				key     = make([]byte, *kl)
				value   = make([]byte, *vl)
				kvreq   = kc.MapRequest{Ct: kc.Write,
					Key:   key,
					Value: value,
				}
			)

			for i := 0; i < *cmds; i++ {
				bgen.GetBytes(key)
				bgen.GetBytes(value)
				kvreq.Marshal(buffer)
				reqsent = time.Now()
				response := <-serviceConnection.Send(buffer.Bytes())
				if response.Err != nil {
					log.Fatalln("Response error:", response.Err)
				}
				elog.Log(e.NewTimedEvent(e.ClientRequestLatency, reqsent))
				buffer.Reset()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
