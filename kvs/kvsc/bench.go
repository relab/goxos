package main

import (
	"log"
	"time"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/kvs/bgen"
	kc "github.com/relab/goxos/kvs/common"
)

var outputFolder string

func runBench() {
	log.Println("KVS Benchmark Client")

	var err error
	if *report {
		if outputFolder, err = generateReportFolder(); err != nil {
			log.Fatalln("Aborted! Reason:", err)
		}
	}

	if err = setupLogging(*report, outputFolder); err != nil {
		log.Fatalln("Aborted! Reason:", err)
	}

	log.Println("Dialing kvs cluster")
	serviceConnection, err := client.Dial(clientConfig)
	if err != nil {
		log.Fatalln("Error dailing cluster:", err)
	}

	reqLatencies := make([][]time.Duration, *runs)
	runDurations := make([]time.Duration, *runs)

	if *prewait > 0 {
		log.Println("Starting in", *prewait)
		time.Sleep(*prewait)
	}

	log.Println("Running...")
	for i := 0; i < *runs; i++ {
		log.Println("Performing run", i)
		runDurations[i], reqLatencies[i], err = performRun(serviceConnection, *cmds)
		if err != nil {
			log.Fatalln("Aborted! Reason:", err)
		}
	}

	if !*noreport {
		log.Println("Generating report...")
		generateReport(runDurations, reqLatencies)
	}

	log.Println("Done!")
}

func performRun(conn client.ServiceConn, nrOfCmds int) (
	duration time.Duration, reqLatencies []time.Duration, err error) {
	reqLatencies = make([]time.Duration, nrOfCmds)
	var reqSent, respRscv time.Time
	key := make([]byte, *kl)
	val := make([]byte, *vl)
	mapreq := kc.MapRequest{Ct: kc.Write,
		Key:   key,
		Value: val,
	}

	start := time.Now()
	for i := 0; i < nrOfCmds; i++ {
		bgen.GetBytes(key)
		bgen.GetBytes(val)
		mapreq.Marshal(buffer)
		reqSent = time.Now()
		response := <-conn.Send(buffer.Bytes())
		respRscv = time.Now()

		// Abort if we get an error response
		if response.Err != nil {
			return 0, nil, response.Err
		}

		reqLatencies[i] = respRscv.Sub(reqSent)
		buffer.Reset()
	}

	end := time.Now()
	duration = end.Sub(start)
	return
}
