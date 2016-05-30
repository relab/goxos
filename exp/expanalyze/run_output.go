package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"time"

	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/exp/util"

	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plot"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plotter"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plotutil"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/vg"
)

const (
	stderrText         = ".stderr.txt"
	glogText           = ".glog.txt"
	elogText           = ".elog.txt"
	throughputPlotFile = "throughput.png"
)

func (r *Run) generateOutput() error {
	if err := r.generateFolders(); err != nil {
		return err
	}
	if err := r.generateLogs(); err != nil {
		return err
	}
	if r.Valid && *cplot {
		if err := r.generateClientLatPlots(); err != nil {
			return err
		}
	}
	if r.Valid && *rplot {
		if err := r.generateReplicaThroughputPlots(); err != nil {
			return err
		}
	}
	if err := r.generateHTML(); err != nil {
		return err
	}
	if r.ID == *csvrun {
		if err := r.dumpCVS(); err != nil {
			return err
		}
	}

	return nil
}

func (r *Run) generateFolders() error {
	return os.MkdirAll(r.getRunRootReport(), 0755)
}

func (r *Run) generateHTML() error {
	b := new(bytes.Buffer)
	expTemplate := template.Must(template.New(templateRun).Parse(templateRun))
	if err := expTemplate.Execute(b, r); err != nil {
		return err
	}

	err := ioutil.WriteFile(path.Join(r.getRunRootReport(), r.ID+".html"), b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *Run) generateLogs() error {
	r.LogURLs = make(map[string]*logURL)
	r.dumpReadableReplicaElogs()
	r.copyGlogs()
	r.copyStderrs()
	return nil
}

func (r *Run) dumpReadableReplicaElogs() {
	for _, replica := range expreport.Hostnames.Nodes {
		if err := e.DumpAsTextFile(path.Join(r.getRunRootReport(), replica+elogText),
			r.ReplicaEvents[replica]); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("dumpElogs: %v", err))
			continue
		}
		lu, found := r.LogURLs[replica]
		if found {
			lu.Elog = replica + elogText
		} else {
			r.LogURLs[replica] = &logURL{Elog: replica + elogText}
		}
	}
	for _, standby := range expreport.Hostnames.Standbys {
		if err := e.DumpAsTextFile(r.getRunRootReport()+"/"+standby+elogText,
			r.ReplicaEvents[standby]); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("dumpElogs: %v", err))
			continue
		}
		lu, found := r.LogURLs[standby]
		if found {
			lu.Elog = standby + elogText
		} else {
			r.LogURLs[standby] = &logURL{Elog: standby + elogText}
		}
	}
}

func (r *Run) copyGlogs() {
	for _, replica := range expreport.Hostnames.Nodes {
		if err := util.Copy(path.Join(r.getRunRootReport(), replica+glogText),
			path.Join(r.getRunRootCollect(), replica, cfg.GetString("glogName", ""))); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("copyGlogs: %v", err))
			continue
		}
		lu, found := r.LogURLs[replica]
		if found {
			lu.Glog = replica + glogText
		} else {
			r.LogURLs[replica] = &logURL{Glog: replica + glogText}
		}
	}
	for _, standby := range expreport.Hostnames.Standbys {
		if err := util.Copy(path.Join(r.getRunRootReport(), standby+glogText),
			path.Join(r.getRunRootCollect(), standby, cfg.GetString("glogName", ""))); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("copyGlogs: %v", err))
			continue
		}
		lu, found := r.LogURLs[standby]
		if found {
			lu.Glog = standby + glogText
		} else {
			r.LogURLs[standby] = &logURL{Glog: standby + glogText}
		}
	}
}

func (r *Run) copyStderrs() {
	for _, replica := range expreport.Hostnames.Nodes {
		if err := util.Copy(path.Join(r.getRunRootReport(), replica+stderrText),
			path.Join(r.getRunRootCollect(), replica, "stderr.txt")); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("copyStderrs: %v", err))
			continue
		}
		lu, found := r.LogURLs[replica]
		if found {
			lu.Stderr = replica + stderrText
		} else {
			r.LogURLs[replica] = &logURL{Stderr: replica + stderrText}
		}
	}
	for _, client := range expreport.Hostnames.Clients {
		if err := util.Copy(path.Join(r.getRunRootReport(), client+stderrText),
			path.Join(r.getRunRootCollect(), client, "stderr.txt")); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("copyStderr: %v", err))
			continue
		}
		lu, found := r.LogURLs[client]
		if found {
			lu.Stderr = client + stderrText
		} else {
			r.LogURLs[client] = &logURL{Stderr: client + stderrText}
		}
	}
	for _, standby := range expreport.Hostnames.Standbys {
		if err := util.Copy(path.Join(r.getRunRootReport(), standby+stderrText),
			path.Join(r.getRunRootCollect(), standby, "stderr.txt")); err != nil {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("copyStderrs: %v", err))
			continue
		}
		lu, found := r.LogURLs[standby]
		if found {
			lu.Stderr = standby + stderrText
		} else {
			r.LogURLs[standby] = &logURL{Stderr: standby + stderrText}
		}
	}
}

const (
	plotHeight     = 11 // ~1059px
	plotInchPerSec = 20 // ~1920px
)

func (r *Run) generateClientLatPlots() error {
	r.ClientPlots = make(map[string]string)

	var (
		err           error
		p             *plot.Plot
		l             *plotter.Line
		medMinMax     *plotutil.ErrorPoints
		clientsToPlot = r.getRandomClients()
	)

	for i, client := range clientsToPlot {
		clatencies, found := r.ClientLatencies[client]
		if !found {
			return fmt.Errorf("plotting: could not find latencies for client %s", client)
		}
		points := convertLatToSecXY(clatencies, r.TimeZero)

		p, err = plot.New()
		if err != nil {
			goto Error
		}

		l, err = plotter.NewLine(points)
		if err != nil {
			goto Error
		}
		l.LineStyle.Width = vg.Points(1)
		l.LineStyle.Color = plotutil.Color((i + 1) % 6)

		p.Add(l)
		p.Legend.Add(client, l)

		medMinMax, err = plotutil.NewErrorPoints(plotutil.MedianAndMinMax, points)
		if err != nil {
			goto Error
		}
		//r.MeanClLatency = medMinMax.XYs[0].Y
		//r.MaxClLatency = medMinMax.YErrors[0].High

		plotutil.AddLinePoints(p, "median and minimum and maximum", medMinMax)
		plotutil.AddErrorBars(p, medMinMax)

		p.Add(plotter.NewGrid())
		p.Legend.Top = true
		p.Legend.Left = true
		p.Title.Text = client
		p.X.Label.Text = "Time (seconds)"
		p.Y.Label.Text = "Request latency (milliseconds)"

		file := path.Join(r.getRunRootReport(), client+".png")
		if err = p.Save(getClientRunDuration(clatencies)*plotInchPerSec,
			plotHeight, file); err != nil {
			goto Error
		}
		r.ClientPlots[client] = client + ".png"
	}

	return nil

Error:
	r.Warnings = append(r.Warnings,
		fmt.Sprintf("Generate client plots error: %v", err))
	return nil
}

func convertLatToSecXY(lats []e.Event, timeZero time.Time) plotter.XYs {
	pts := make(plotter.XYs, len(lats))
	for i, lat := range lats {
		pts[i].X = (lat.EndTime.Sub(timeZero)).Seconds()
		pts[i].Y = float64(lat.EndTime.Sub(lat.Time)) / float64(time.Millisecond)
	}

	return pts
}

func (r *Run) getRandomClients() []string {
	nclients := len(r.ClientLatencies)
	if *cplots == 0 || nclients == 0 {
		return []string{}
	}

	clients := make([]string, nclients)

	var i int
	for client := range r.ClientLatencies {
		clients[i] = client
		i++
	}

	// Return all
	if int(*cplots) >= nclients {
		return clients
	}

	// Pick # random
	rand.Seed(time.Now().Unix())
	for i := range clients {
		j := rand.Intn(i + 1)
		clients[i], clients[j] = clients[j], clients[i]
	}

	return clients[:*cplots]
}

func convertThroughputToSecXY(samples []e.Event, timeZero time.Time) plotter.XYs {
	pts := make(plotter.XYs, len(samples))
	for i, s := range samples {
		pts[i].X = (s.Time.Sub(timeZero)).Seconds()
		pts[i].Y = float64(s.Value)
	}

	return pts
}

func (r *Run) generateReplicaThroughputPlots() error {
	file := path.Join(r.getRunRootReport(), throughputPlotFile)

	var (
		err    error
		runDur float64
		p      *plot.Plot
		l      *plotter.Line
		i      int
	)

	p, err = plot.New()
	if err != nil {
		goto Error
	}

	p.Add(plotter.NewGrid())
	p.Title.Text = "Replica throughput"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Throughput"

	for replica, samples := range r.ReplicaThroughput {
		points := convertThroughputToSecXY(samples, r.TimeZero)
		l, err = plotter.NewLine(points)
		if err != nil {
			goto Error
		}
		l.LineStyle.Width = vg.Points(2)
		l.LineStyle.Dashes = plotutil.Dashes(i)
		l.LineStyle.Color = plotutil.Color(i)
		p.Add(l)
		p.Legend.Add(replica, l)
		p.Legend.Top = true
		p.Legend.Left = true
		i++
	}

	runDur, err = getReplicaRunDuration(r.ReplicaThroughput, failures)
	if err != nil {
		return err
	}

	if err = p.Save(runDur*plotInchPerSec, plotHeight, file); err != nil {
		goto Error
	}

	r.ThroughputPlot = throughputPlotFile
	return nil

Error:
	r.Warnings = append(r.Warnings,
		fmt.Sprintf("Generate replica throughput plots error: %v", err))
	return nil
}

func getClientRunDuration(events []e.Event) (seconds float64) {
	if events == nil {
		panic("getDurationOfEvents: got nil slice")
	}
	if len(events) == 0 || len(events) == 1 {
		return 0
	}

	return (events[len(events)-1].EndTime.Sub(events[0].EndTime)).Seconds()
}

func getReplicaRunDuration(rsamples map[string][]e.Event,
	failureMap map[string]bool) (seconds float64, err error) {
	if rsamples == nil {
		return 0, errors.New("getReplicaRunDuration: got rsamples nil map")
	}
	if failureMap == nil {
		return 0, errors.New("getReplicaRunDuration: got failure nil map")
	}

	for replica, samples := range rsamples {
		if failureMap[replica] {
			continue
		}
		if len(samples) == 0 || len(samples) == 1 {
			return 0, nil
		}

		return (samples[len(samples)-1].Time.Sub(samples[0].Time)).Seconds(), nil
	}

	return 0, errors.New("getReplicaRunDuration: unable to calculate duration from data")
}

func (r *Run) dumpCVS() error {
	// Events standby
	// Scan for CU
	var cus, cud e.Event
	standbyElog, _ := r.ReplicaEvents[expreport.Hostnames.Standbys[0]]
	for _, event := range standbyElog {
		if event.Type == e.CatchUpMakeReq {
			cus = event
			break
		}
	}
	for i := len(standbyElog) - 1; i >= 0; i-- {
		if standbyElog[i].Type == e.CatchUpDoneHandlingResp {
			cud = standbyElog[i]
			break
		}
	}
	if cus.Time.IsZero() || cud.Time.IsZero() {
		panic("Did not find CU")
	}
	buffer := new(bytes.Buffer)
	writer := csv.NewWriter(buffer)
	writer.Write([]string{"InitStart", "InitDone", "ActivDone", "CuStart", "CuDone"})
	writer.Write([]string{
		fmt.Sprintf("%f", r.is.Time.Sub(r.TimeZero).Seconds()),
		fmt.Sprintf("%f", r.id.Time.Sub(r.TimeZero).Seconds()),
		fmt.Sprintf("%f", r.ad.Time.Sub(r.TimeZero).Seconds()),
		fmt.Sprintf("%f", cus.Time.Sub(r.TimeZero).Seconds()),
		fmt.Sprintf("%f", cud.Time.Sub(r.TimeZero).Seconds()),
	})
	writer.Flush()
	if err := writer.Error(); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("events.csv", buffer.Bytes(), 0644); err != nil {
		panic(err)
	}

	// Replica through
	for replica, tsamples := range r.ReplicaThroughput {
		if failures[replica] {
			continue
		}
		buffer = new(bytes.Buffer)
		writer = csv.NewWriter(buffer)
		writer.Write([]string{"Time", "Throughput"})
		var csvSamples []e.Event
		if replica == expreport.Hostnames.Standbys[0] {
			csvSamples = tsamples[:10]
			goto Write
		}
		for i, s := range tsamples {
			if s.Time.After(r.is.Time) {
				csvSamples = tsamples[i-5:]
				break
			}
		}

		for j := len(csvSamples) - 1; j >= 0; j-- {
			if csvSamples[j].Time.Before(r.ad.Time) {
				csvSamples = csvSamples[:j+14]
				break
			}
		}
	Write:
		for _, s := range csvSamples {
			writer.Write([]string{
				fmt.Sprintf("%f", s.Time.Sub(r.TimeZero).Seconds()),
				fmt.Sprintf("%d", s.Value*uint64(tsc)),
			})
			if err := writer.Error(); err != nil {
				panic(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(replica+".csv", buffer.Bytes(), 0644); err != nil {
			panic(err)
		}
	}

	// Client lats
	for client, csamples := range r.ClientLatencies {
		buffer = new(bytes.Buffer)
		writer = csv.NewWriter(buffer)
		writer.Write([]string{"Time", "RTT"})
		var csvSamples []e.Event
		for i, s := range csamples {
			if s.Time.After(r.is.Time.Add(-100 * time.Millisecond)) {
				csvSamples = csamples[i:]
				break
			}
		}
		for j := len(csvSamples) - 1; j >= 0; j-- {
			if csvSamples[j].Time.Before(r.ad.Time.Add(300 * time.Millisecond)) {
				csvSamples = csvSamples[:j]
				break
			}
		}
		for _, s := range csvSamples {
			writer.Write([]string{
				fmt.Sprintf("%f", s.Time.Sub(r.TimeZero).Seconds()),
				fmt.Sprintf("%f", float64(s.EndTime.Sub(s.Time))/float64(time.Millisecond)),
			})
			if err := writer.Error(); err != nil {
				panic(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(client+".csv", buffer.Bytes(), 0644); err != nil {
			panic(err)
		}
	}

	return nil
}
