package main

import (
	"fmt"
	"os"

	"github.com/relab/goxos/client"
	kc "github.com/relab/goxos/kvs/common"
)

func runUser() {
	printHeader()

	fmt.Println("Dialing kvs cluster...")
	serviceConnection, err := client.Dial(clientConfig)
	if err != nil {
		fmt.Println("Error dailing cluster:", err)
		os.Exit(1)
	}

	var (
		req        kc.MapRequest
		resp       kc.MapResponse
		ct         int
		key, value string
	)

	for {
		fmt.Println("\nEnter command type (int):")
		for i, ct := range kc.Commandtypes {
			fmt.Printf("%v for %q\n", i+1, ct)
		}

		fmt.Scanln(&ct)

		switch ct {
		case 1, 3:
			fmt.Println("Enter key:")
			fmt.Scanln(&key)
		case 2:
			fmt.Println("Enter key:")
			fmt.Scanln(&key)
			fmt.Println("Enter value:")
			fmt.Scanln(&value)
		default:
			fmt.Println("Error: Unkown command")
			continue
		}

		req = kc.MapRequest{
			Ct:    kc.CommandType(ct - 1),
			Key:   []byte(key),
			Value: []byte(value),
		}

		buffer.Reset()
		req.Marshal(buffer)

		fmt.Println(req)

		response := <-serviceConnection.Send(buffer.Bytes())
		if response.Err != nil {
			fmt.Println("Error when sending request: ", response.Err)
			continue
		}

		buffer.Reset()
		buffer.Write(response.Value)

		if err = resp.Unmarshal(buffer); err != nil {
			fmt.Println("Unmarshal error:, err")
			continue
		}

		fmt.Println(resp)
	}
}

func printHeader() {
	fmt.Println("------------------------")
	fmt.Println("KEY-VALUE STORE CLIENT  ")
	fmt.Println("------------------------")
	fmt.Println("")
	fmt.Println("Info: Key is a string.")
	fmt.Println("Info: Value is a string.")
	fmt.Println("Info: Control-C to exit.")
	fmt.Println("")
}
