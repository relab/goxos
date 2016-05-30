package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/relab/goxos/exp/util"
)

const defaultHosts = `pitter21 pitter22 pitter23 pitter24 pitter25
pitter26 pitter27 pitter28 pitter29 pitter30 pitter31 pitter32
pitter33 pitter34 pitter35 pitter36 pitter37`

const usage = `

Reads a experiment ini file, pings all hostnames (replicas, standbys
and clients) and outputs a new experiment ini where not-available
hosts are replaced with others available.

Logs progress to stderr, and (when no -w flag) dumps the new ini-file
to stdout.

Usage:

	exppinghosts [-w] [-a=<path>] -exp_path=<exp-ini-file>

Arguments:

	w	Writes the changes back to the file instead of printing.
	a       File with whitespace-separated hostnames of alternatives.

The a-argument can be omitted. It will then use this default list:
	pitter21, pitter22, pitter23, pitter24, pitter25, pitter26,
	pitter27, pitter28, pitter29, pitter30, pitter31, pitter32,
	pitter33, pitter34, pitter35, pitter36, pitter37

Example:
exppinghosts -w -exp_path=exprun/example-exp.ini
`

func main() {
	exppath := flag.String("exp_path", "", "Path to experiment ini file")
	writeback := flag.Bool("w", false, "Writeback?")
	available := flag.String("a", "", "Path to file with available hostnames")

	flag.Usage = func() {
		log.Fatal(usage)
		os.Exit(2)
	}

	flag.Parse()

	//
	// Read experiment file
	//
	if *exppath == "" {
		log.Fatal(usage)
		os.Exit(2)
	}

	expini, err := util.LoadFile(*exppath)
	if err != nil {
		log.Fatalln(err)
	}

	var replicas []string
	if val, found := expini.Get("exp", "replicas"); found {
		for _, node := range strings.Split(val, ",") {
			if len(node) > 0 {
				replicas = append(replicas, strings.TrimSpace(node))
			}
		}
	}

	var standbys []string
	if val, found := expini.Get("exp", "standbys"); found {
		for _, node := range strings.Split(val, ",") {
			if len(node) > 0 {
				standbys = append(standbys, strings.TrimSpace(node))
			}
		}
	}

	var clients []string
	if val, found := expini.Get("exp", "clients"); found {
		for _, node := range strings.Split(val, ",") {
			if len(node) > 0 {
				clients = append(clients, strings.TrimSpace(node))
			}
		}
	}

	//
	// Ping hostnames
	//
	var replace []string
	replace = append(replace, ping(replicas, true)...)
	replace = append(replace, ping(standbys, true)...)
	replace = append(replace, ping(clients, true)...)

	if len(replace) == 0 {
		log.Println("All hosts are up and running!")

		if !*writeback {
			expfile, err := ioutil.ReadFile(*exppath)
			if err != nil {
				log.Fatalln(err)
			}

			fmt.Println(string(expfile))
		}
		os.Exit(0)
	}

	//
	// Read alternatives and ping
	//
	alternatives := ""

	if *available == "" {
		alternatives = defaultHosts
	} else {
		alt, err := ioutil.ReadFile(*available)
		if err != nil {
			log.Fatalln(err)
		}
		alternatives = string(alt)
	}

	// Remove already used hosts from the alternatives:
	var alternativeHosts []string
	usedHosts := append(replicas, standbys...)
	usedHosts = append(usedHosts, clients...)
	for _, host := range strings.Fields(alternatives) {
		isUsed := false
		for _, used := range usedHosts {
			if host == used {
				isUsed = true
			}
		}
		if !isUsed {
			alternativeHosts = append(alternativeHosts, host)
		}
	}

	// Get which of the alternatives that are available:
	availableHosts := ping(alternativeHosts, false)

	if len(availableHosts) < len(replace) {
		log.Fatal("Not enough available hosts! Need ", (len(replace) - len(availableHosts)), " more")
		os.Exit(1)
	}

	//
	// Replace hostnames in config:
	//
	expfile, err := ioutil.ReadFile(*exppath)
	if err != nil {
		log.Fatalln(err)
	}

	output := string(expfile)
	for i, host := range replace {
		output = strings.Replace(output, host, availableHosts[i], -1)
	}

	//
	// Write result
	//
	if *writeback {
		err = ioutil.WriteFile(*exppath, ([]byte)(output), 0777)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Done! Replaced", len(replace), "hosts")
	} else {
		fmt.Println(output)
	}
}

// Pings given hosts in parallel and returns list of available or
// unavailable hosts.
func ping(hosts []string, returnUnavailable bool) []string {
	var toReturn []string
	var cmds []*exec.Cmd

	// Start pinging:
	for _, host := range hosts {
		log.Println("Pinging", host)
		// cmd := exec.Command("ssh", "-o ConnectTimeout=1", host, "echo")
		cmd := exec.Command("nc", "-z", "-w 1", host, "22")
		cmd.Start()
		cmds = append(cmds, cmd)
	}

	// Read result of the pings:
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			log.Println("Unavailable host:", hosts[i], "ping error:", err)
			if returnUnavailable {
				toReturn = append(toReturn, hosts[i])
			}
		} else {
			if !returnUnavailable {
				toReturn = append(toReturn, hosts[i])
			}
		}
	}

	return toReturn
}
