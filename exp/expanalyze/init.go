package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/exp/util"
)

const (
	expDescINI      = "exp.ini"
	reportFolder    = "report"
	timestampFormat = "Mon Jan 2 15:04:05 2006"
)

func initialize() error {
	if err := readExperimentDescription(); err != nil {
		return err
	}
	if err := initReport(); err != nil {
		return err
	}
	if err := checkLogRoot(); err != nil {
		return err
	}
	if err := genReportFolder(); err != nil {
		return err
	}

	createFailureMap()
	setGenStats()
	setTsc()

	return nil
}

func readExperimentDescription() error {
	ini, err := util.LoadFile(path.Join(exproot, expDescINI))
	if err != nil {
		return fmt.Errorf("readExperimentDescription: %v", err)
	}

	cfg = *config.NewConfig()
	for k, v := range ini.Section("exp") {
		cfg.Set(k, v)
	}
	for k, v := range ini.Section("goxos") {
		cfg.Set(k, v)
	}

	return nil
}

func initReport() error {
	nodes := []string{}
	for _, node := range strings.Split(cfg.GetString("replicas", ""), ",") {
		if len(strings.TrimSpace(node)) > 0 {
			nodes = append(nodes, strings.TrimSpace(node))
		}
	}

	standbys := []string{}
	for _, standby := range strings.Split(cfg.GetString("standbys", ""), ",") {
		if len(strings.TrimSpace(standby)) > 0 {
			standbys = append(standbys, strings.TrimSpace(standby))
		}
	}

	clients := []string{}
	for _, client := range strings.Split(cfg.GetString("clients", ""), ",") {
		if len(strings.TrimSpace(client)) > 0 {
			clients = append(clients, strings.TrimSpace(client))
		}
	}

	failures := []string{}
	for _, failure := range strings.Split(cfg.GetString("failures", ""), ",") {
		if len(strings.TrimSpace(failure)) > 0 {
			failures = append(failures, strings.TrimSpace(strings.Split(failure, ":")[1]))
		}
	}

	expreport = &ExpReport{
		Cfg:     cfg,
		ExpRuns: make([]*Run, cfg.GetInt("runs", 0)),
		Hostnames: Hostnames{
			Nodes:    nodes,
			Standbys: standbys,
			Clients:  clients,
			Failures: failures,
		},
	}

	// read Go version and goxos/c/apps commits
	v, err := readVersions(path.Join(exproot, cfg.GetString("localVersOut", "")))
	if err != nil {
		return fmt.Errorf("initReport: %v", err)
	}

	// read timestamp from file
	t, err := parseTimestamp(path.Join(exproot, cfg.GetString("localTimeOut", "")))
	if err != nil {
		return fmt.Errorf("initReport: %v", err)
	}

	// set output flags
	of := setOutputFlags()

	expreport.Versions = v
	expreport.Timestamp = t
	expreport.OutputFlags = of

	return nil
}

func readVersions(filename string) (Versions, error) {
	vers, err := ioutil.ReadFile(filename)
	if err != nil {
		return Versions{}, fmt.Errorf("readVersions: %v", err)
	}

	var v Versions
	b := bytes.NewBuffer(vers)
	reader := bufio.NewReader(b)

	v.Go, err = reader.ReadString('\n')
	if err != nil {
		v.Go = "Unknown"
	}

	v.Goxos, err = reader.ReadString('\n')
	if err != nil {
		v.Goxos = "Unknown"
	}

	return v, nil
}

func parseTimestamp(filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("parseTimestamp: %v", err)
	}

	// Cloud also use something like bufio and ReadString...
	tss := strings.TrimSuffix(string(b), "\n")
	tsf, err := strconv.ParseFloat(tss, 32)
	if err != nil {
		return "", fmt.Errorf("parseTimestamp: %v", err)
	}

	timestamp := time.Unix(int64(tsf), 0)
	return timestamp.Format(timestampFormat), nil
}

func checkLogRoot() error {
	if exists, _ := util.Exists(path.Join(exproot, cfg.GetString("localCollectDir", ""))); !exists {
		return fmt.Errorf("could not find experiment log directory")
	}

	return nil
}

func genReportFolder() error {
	if err := os.MkdirAll(path.Join(exproot, reportFolder), 0755); err != nil {
		return err
	}

	return nil
}

func createFailureMap() {
	failures = make(map[string]bool)
	for _, failure := range strings.Split(cfg.GetString("failures", ""), ",") {
		if len(strings.TrimSpace(failure)) > 0 {
			failures[strings.TrimSpace(strings.Split(failure, ":")[1])] = true
		}
	}
}

func setGenStats() {
	genstats = len(failures) > 0 && len(strings.TrimSpace(cfg.GetString("standbys", ""))) > 0
}

func setTsc() {
	tsc = float64(1000 / int64(
		cfg.GetDuration(
			"throughputSamplingInterval",
			time.Duration(25*time.Millisecond),
		)/time.Millisecond),
	)
}

func setOutputFlags() OutputFlags {
	of := OutputFlags{}
	switch cfg.GetString("failureHandlingType", "None") {
	case "LiveReplacement":
		of.LREnabled = true
	case "AsynchronousReconfiguration":
		of.ARecEnabled = true
	}

	return of
}
