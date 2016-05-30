package main

import (
	"log"
	"time"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/kvs/bgen"
	kc "github.com/relab/goxos/kvs/common"
)

func runBenchAsync() {
	log.Println("KVS Async Benchmark Client")

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

	if *prewait > 0 {
		log.Println("Starting in", *prewait)
		time.Sleep(*prewait)
	}

	log.Println("Running...")
	for i := 0; i < *runs; i++ {
		log.Println("Performing run", i)
		reqLatencies[i], err = performAsyncRun(serviceConnection, *cmds)
		if err != nil {
			log.Fatalln("Aborted! Reason:", err)
		}
	}

	if !*noreport {
		log.Println("Generating report...")
		generateReport(nil, reqLatencies)
	}

	log.Print("Responses not received: ", countResponsesNotReceived(reqLatencies))

	log.Println("Done!")
}

func countResponsesNotReceived(times [][]time.Duration) int {
	counter := 0
	for i := range times {
		for j := range times[i] {
			if !(times[i][j] > 0) {
				counter++
			}
		}
	}
	return counter
}

func performAsyncRun(conn client.ServiceConn, nrOfCmds int) ([]time.Duration, error) {
	key := make([]byte, *kl)
	val := make([]byte, *vl)
	mapreq := kc.MapRequest{Ct: kc.Write,
		Key:   key,
		Value: val,
	}

	responseChannels := make([]<-chan client.ResponseData, nrOfCmds)

	for i := 0; i < nrOfCmds; i++ {
		//log.Println("Sending command number: ", i)
		bgen.GetBytes(key)
		bgen.GetBytes(val)
		mapreq.Marshal(buffer)

		responseChannels[i] = conn.Send(buffer.Bytes())

		// time.Sleep(10 * time.Millisecond)

		buffer.Reset()
	}

	log.Println("Waiting for responses")
	reqLatencies, err := getRequestLatencies(responseChannels)

	return reqLatencies, err
}

func getRequestLatencies(responseChannels []<-chan client.ResponseData) ([]time.Duration, error) {
	//<-time.After(2 * time.Second)
	reqLatencies := make([]time.Duration, len(responseChannels))

	for index, responseChan := range responseChannels {
	readloop:
		for readFromChannelAttempts := 0; readFromChannelAttempts < 5; readFromChannelAttempts++ {
			select {
			case response := <-responseChan:
				if response.Err != nil {
					return nil, response.Err
				}
				reqLatencies[index] = response.ReceiveTime.Sub(response.SendTime)
				//log.Printf("Response received: %v", response)
				break readloop
			default:
				//log.Printf("Waiting 10 ms")
				<-time.After(10 * time.Millisecond)
			}
		}
	}
	return reqLatencies, nil
}
