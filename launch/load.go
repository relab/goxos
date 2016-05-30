package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

// load paths ...
//   Load the specified configuration files or directories of configuration
//   files. At least one file or directory must be provided. We do not
//   traverse subdirectories.

const loadName = "load"

var loadCmd = cli.Command{
	Name:      loadName,
	ShortName: "ld",
	Usage:     "load the specified configuration files or directories of configuration files",
	Action:    load,
}

// The load function is only for parsing configuration files/directories. We
// rely on the submitService and its corresponding submitHandler, and thus we
// do not need to have separate loadService/Handler functions.
func load(c *cli.Context) {
	args := c.Args()
	if len(args) == 0 {
		fmt.Println("No file or directory provided.")
		cli.ShowCommandHelp(c, loadName)
		os.Exit(-2)
	}
	loadAndSubmit(c.Args())
}

func loadAndSubmit(paths []string) {
	for _, path := range paths {
		services := loadConfig(path)
		for _, v := range services {
			log.Println(v)
		}
		submitServices(services)
	}
}

func loadConfig(path string) map[string]*ServiceInfo {
	log.Printf("loading configuration from %s ...\n", path)
	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	services := make(map[string]*ServiceInfo)
	if fi.IsDir() {
		cfgFiles := filter(filesIn(path), func(file string) bool {
			return strings.HasSuffix(file, fileType)
		})
		for _, file := range cfgFiles {
			srvInfo := readConfigFile(filepath.Join(path, file))
			services[srvInfo.Label] = &srvInfo
		}
	} else {
		srvInfo := readConfigFile(path)
		services[srvInfo.Label] = &srvInfo
	}
	return services
}

func readConfigFile(path string) ServiceInfo {
	fd, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(fd)
	var s ServiceInfo
	if err = dec.Decode(&s); err != nil {
		panic(err)
	}
	return s
}

// filter returns a new slice with elements of s that satisfy fn()
func filter(s []string, fn func(string) bool) []string {
	var p []string
	for _, v := range s {
		if fn(v) {
			p = append(p, v)
		}
	}
	return p
}

// filesIn returns files in the given directory dir
func filesIn(dir string) []string {
	fh, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	files, err := fh.Readdirnames(0)
	if err != nil {
		panic(err)
	}
	return files
}
