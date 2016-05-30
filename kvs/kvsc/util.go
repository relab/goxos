package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func setupLogging(writeToFile bool, outputFolder string) error {
	if writeToFile {
		logfile, err := os.OpenFile(outputFolder+"/report.txt",
			os.O_WRONLY|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	}
	return nil
}

func generateReportFolder() (string, error) {
	folderName := renderTimestamp(time.Now())
	if err := os.Mkdir(folderName, 0755); err != nil {
		return folderName, err
	}
	return folderName, nil
}

func logBenchmarkProperties() {
	log.Println("Time:", time.Now())
	log.Println("Number of runs to be done:", *runs)
	log.Println("Number of clients:", *nclients)
	log.Println("Number of commands per run:", *cmds)
	log.Println("Key length in bytes:", *kl)
	log.Println("Value length in bytes:", *vl)
	log.Println("Save report:", *report)
}

func renderTimestamp(t time.Time) string {
	return fmt.Sprintf("%d%02d%02d-%02d%02d%02d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second())
}
