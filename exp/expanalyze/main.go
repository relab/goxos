package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/exp/expjsonreport"
	"github.com/relab/goxos/exp/util"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/pkg/math"
)

var (
	workerGoroutines int             // number of worker goroutines for parsing/analysis/output
	exproot          string          // root experiment folder
	cfg              config.Config   // experiment description form INI
	expreport        *ExpReport      // used to save results and generate output
	genstats         bool            // should we calculate statistics for this experiment
	tsc              float64         // throughput req/sampling-interval to req/sec constant
	failures         map[string]bool // utility failure map
)

const usage = `Usage:

	expanalyse -path <path-to-experiment-folder> [-rplot -cplot -cplots x]

example:
expanalyse -path ExperimentFoo-20131129-104249 -cplots 3
`

var (
	exppath = flag.String("path", "", "Path to root experiment folder")
	rplot   = flag.Bool("rplot", false, "Enable plotting of replica throughput per run")
	cplot   = flag.Bool("cplot", false, "Enable plotting of client latencies per run")
	cplots  = flag.Uint("cplots", 1, "Number of clients plots to be generated per run")
	csvrun  = flag.String("csvrun", "-1", "Run to dump as CSV for plotting")
	clatana = flag.Bool("clatana", false, "Analyze Client Latencies")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage, "\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if *exppath == "" {
		flag.Usage()
		os.Exit(1)
	}

	exproot = *exppath
	runtime.GOMAXPROCS(runtime.NumCPU())
	workerGoroutines = runtime.NumCPU()

	if err := initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Initalization failed: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	parseRuns()
	analyseRuns()
	calcLatencies()
	analyseExperiment()
	generateOutput()
	generateJSONReport()
}

func parseRuns() {
	// Parse runs in parallel
	parseRunNumber := make(chan int, 16)
	var wg sync.WaitGroup
	for i := 0; i < workerGoroutines; i++ {
		wg.Add(1)
		go func() {
			for i := range parseRunNumber {
				expreport.ExpRuns[i] = ParseRun(i)
			}
			wg.Done()
		}()
	}

	for i := 0; i < cfg.GetInt("runs", 0); i++ {
		parseRunNumber <- i
	}
	close(parseRunNumber)
	wg.Wait()
}

func analyseRuns() {
	// Analyse runs in parallel
	analyseRuns := make(chan *Run, 16)
	var wg sync.WaitGroup
	for i := 0; i < workerGoroutines; i++ {
		wg.Add(1)
		go func() {
			for run := range analyseRuns {
				run.Analyse(genstats)
			}
			wg.Done()
		}()
	}

	for _, run := range expreport.ExpRuns {
		analyseRuns <- run
	}
	close(analyseRuns)
	wg.Wait()
}

func calcLatencies() {
	// Analyse runs in parallel
	if !*clatana {
		return
	}
	calcLatencies := make(chan *Run, 16)
	var wg sync.WaitGroup
	for i := 0; i < workerGoroutines; i++ {
		wg.Add(1)
		go func() {
			for run := range calcLatencies {
				run.calcLatency()
			}
			wg.Done()
		}()
	}

	for _, run := range expreport.ExpRuns {
		if run.Valid {
			calcLatencies <- run
		}
	}
	close(calcLatencies)
	wg.Wait()
}

func analyseExperiment() {
	if validRunsPresent() && genstats {
		expreport.Analysed = true
		analyseDurations()
		analyseThroughput()
		analyseLatencies()
	}
	analyseTotalAverageThroughput()
}

func validRunsPresent() bool {
	for _, run := range expreport.ExpRuns {
		if run.Valid {
			expreport.ExpSummary.ValidRuns++
		}
	}
	if expreport.ExpSummary.ValidRuns == 0 {
		expreport.ExpSummary.Comment = "No valid run to analyse"
		return false
	}

	return true
}

func analyseDurations() {
	initDurs := make([]time.Duration, expreport.ExpSummary.ValidRuns)
	actiDurs := make([]time.Duration, expreport.ExpSummary.ValidRuns)
	fAccDurs := make([]time.Duration, expreport.ExpSummary.ValidRuns)
	arecActiDurs := make([]time.Duration, expreport.ExpSummary.ValidRuns)

	i := 0
	for _, run := range expreport.ExpRuns {
		if run.Valid {
			initDurs[i] = run.InitDur
			actiDurs[i] = run.ActiDur
			fAccDurs[i] = run.FirstAcc
			arecActiDurs[i] = run.ARecActiDur
			i++
		}
	}

	expreport.ExpSummary.InitDurMean = util.MeanDuration(initDurs...)
	expreport.ExpSummary.InitDurSD = util.SSDDuration(initDurs...)
	expreport.ExpSummary.InitDurSdOfMean = util.StdErrMeanDuration(expreport.ExpSummary.InitDurSD,
		expreport.ExpSummary.ValidRuns)
	expreport.ExpSummary.InitDurMin = util.MinDurationN(initDurs...)
	expreport.ExpSummary.InitDurMax = util.MaxDurationN(initDurs...)

	expreport.ExpSummary.ActiDurMean = util.MeanDuration(actiDurs...)
	expreport.ExpSummary.ActiDurSD = util.SSDDuration(actiDurs...)
	expreport.ExpSummary.ActiDurSdOfMean = util.StdErrMeanDuration(expreport.ExpSummary.ActiDurSD,
		expreport.ExpSummary.ValidRuns)
	expreport.ExpSummary.ActiDurMin = util.MinDurationN(actiDurs...)
	expreport.ExpSummary.ActiDurMax = util.MaxDurationN(actiDurs...)

	expreport.ExpSummary.FAccDurMean = util.MeanDuration(fAccDurs...)
	expreport.ExpSummary.FAccDurSD = util.SSDDuration(fAccDurs...)
	expreport.ExpSummary.FAccDurSdOfMean = util.StdErrMeanDuration(expreport.ExpSummary.FAccDurSD,
		expreport.ExpSummary.ValidRuns)
	expreport.ExpSummary.FAccDurMin = util.MinDurationN(fAccDurs...)
	expreport.ExpSummary.FAccDurMax = util.MaxDurationN(fAccDurs...)

	if !expreport.ARecEnabled {
		return
	}

	expreport.ExpSummary.ARecActiDurMean = util.MeanDuration(arecActiDurs...)
	expreport.ExpSummary.ARecActiDurSD = util.SSDDuration(arecActiDurs...)
	expreport.ExpSummary.ARecActiDurSdOfMean = util.StdErrMeanDuration(expreport.ExpSummary.ActiDurSD,
		expreport.ExpSummary.ValidRuns)
	expreport.ExpSummary.ARecActiDurMin = util.MinDurationN(arecActiDurs...)
	expreport.ExpSummary.ARecActiDurMax = util.MaxDurationN(arecActiDurs...)
}

func analyseThroughput() {
	var baseTputVals []uint64
	var initTputVals []uint64
	var actiTputVals []uint64

	i := 0
	for j, run := range expreport.ExpRuns {
		// j != 0: Always drop run 0 from total experiment statistics
		if run.Valid && j != 0 {
			baseTputVals = append(baseTputVals, run.BaseThroughSamples...)
			initTputVals = append(initTputVals, run.InitThroughSamples...)
			actiTputVals = append(actiTputVals, run.ActiThroughSamples...)
			i++
		}
	}

	if len(baseTputVals) > 0 {
		expreport.BaseTputMean = util.MeanUint64(baseTputVals...)
		expreport.BaseTputSSD = util.SSDUint64(baseTputVals...)
		expreport.BaseTputStdErrOfMean = util.StdErrMeanFloat(expreport.BaseTputSSD, len(baseTputVals))
		expreport.BaseTputMin = math.MinUint64N(baseTputVals...)
		expreport.BaseTputMax = math.MaxUint64N(baseTputVals...)
		// Adjust results to req/second
		expreport.BaseTputMean *= tsc
		expreport.BaseTputSSD *= tsc
		expreport.BaseTputStdErrOfMean *= tsc
		expreport.BaseTputMin *= uint64(tsc)
		expreport.BaseTputMax *= uint64(tsc)
	}

	if len(initTputVals) > 0 {
		expreport.InitTputMean = util.MeanUint64(initTputVals...)
		expreport.InitTputSSD = util.SSDUint64(initTputVals...)
		expreport.InitTputStdErrOfMean = util.StdErrMeanFloat(expreport.InitTputSSD, len(initTputVals))
		expreport.InitTputMin = math.MinUint64N(initTputVals...)
		expreport.InitTputMax = math.MaxUint64N(initTputVals...)
		// Adjust results to req/second
		expreport.InitTputMean *= tsc
		expreport.InitTputSSD *= tsc
		expreport.InitTputStdErrOfMean *= tsc
		expreport.InitTputMin *= uint64(tsc)
		expreport.InitTputMax *= uint64(tsc)
	}

	if len(actiTputVals) > 0 {
		expreport.ActiTputMean = util.MeanUint64(actiTputVals...)
		expreport.ActiTputSSD = util.SSDUint64(actiTputVals...)
		expreport.ActiTputStdErrOfMean = util.StdErrMeanFloat(expreport.ActiTputSSD, len(actiTputVals))
		expreport.ActiTputMin = math.MinUint64N(actiTputVals...)
		expreport.ActiTputMax = math.MaxUint64N(actiTputVals...)
		// Adjust results to req/second
		expreport.ActiTputMean *= tsc
		expreport.ActiTputSSD *= tsc
		expreport.ActiTputStdErrOfMean *= tsc
		expreport.ActiTputMin *= uint64(tsc)
		expreport.ActiTputMax *= uint64(tsc)
	}
}

func generateOutput() {
	if err := generateIndex(); err != nil {
		log.Fatalln(err)
	}

	// Generate output in parallel
	generateRuns := make(chan *Run, 16)
	var wg sync.WaitGroup
	for i := 0; i < workerGoroutines; i++ {
		wg.Add(1)
		go func() {
			for run := range generateRuns {
				if err := run.generateOutput(); err != nil {
					log.Fatalf("generateRuns: run %s: %v", run.ID, err)
				}
			}
			wg.Done()
		}()
	}

	for _, run := range expreport.ExpRuns {
		generateRuns <- run
	}
	close(generateRuns)
	wg.Wait()
}

func analyseTotalAverageThroughput() {
	var throughputValues []float64

	for j, run := range expreport.ExpRuns {
		// j != 0: Always drop run 0 from total experiment statistics
		if run.Valid && j != 0 {
			throughputValues = append(throughputValues, run.AverageThroughput)
		}
	}
	if len(throughputValues) > 0 {
		expreport.TotalAverageThroughput = util.MeanFloat64(throughputValues...)
	}
}

func analyseLatencies() {
	if !*clatana {
		return
	}
	var maxLats, maxActs []float64
	var meanL, lowerr, higherr []float64

	for j, run := range expreport.ExpRuns {
		//drop run 0
		if run.Valid && j != 0 {
			maxLats = append(maxLats, run.MaxClLatency)
			meanL = append(meanL, run.MedianClLatency)
			lowerr = append(lowerr, run.Low90ErrClLatency)
			higherr = append(higherr, run.High90ErrClLatency)
			if run.MaxActClLatency != 0 {
				maxActs = append(maxActs, run.MaxActClLatency)
			}
		}
	}

	if len(maxLats) > 0 {
		expreport.MeanLatency = util.MedianFloatSlice(meanL)
		expreport.MeanLatMin90 = util.MedianFloatSlice(lowerr)
		expreport.MeanLatMax90 = util.MedianFloatSlice(higherr)
		expreport.MeanMaxLatency = util.MedianFloatSlice(maxLats)
		expreport.MeanMaxActLatency, expreport.MMALMin90, expreport.MMALMax90 = util.MedianMinMax90FloatSlice(maxActs)
	}
}

func generateJSONReport() {
	report := expjsonreport.JSONReport{}
	report.NumberOfValidRuns = expreport.ExpSummary.ValidRuns
	report.TotalNumberOfRuns = cfg.GetInt("runs", 0)
	report.Timestamp = expreport.Timestamp
	report.ExperimentName = cfg.GetString("name", "Unkown")
	report.TotalAverageThroughput = expreport.TotalAverageThroughput
	report.PercentageOfRequestsDecided = getPercentageOfRequestsDecided()
	bytes, err := json.Marshal(report)
	if err != nil {
		log.Println("generateJSONReport: failed to marshal JSON: ", err)
		return
	}

	err = ioutil.WriteFile(path.Join(exproot, reportFolder, "jsonReport.json"), bytes, 0644)
	if err != nil {
		log.Println("generateJSONReport: failed writing json file: ", err)
	}
}

func getPercentageOfRequestsDecided() float64 {
	var decidedRequestsCounter uint64

	for j, run := range expreport.ExpRuns {
		// j != 0: Always drop run 0 from total experiment statistics
		if run.Valid && j != 0 {
			decidedRequestsCounter += run.RequestsDecided
		}
	}
	if decidedRequestsCounter > 0 {
		expectedDecidedRequests := (cfg.GetInt("runs", 0) - 1) *
			len(expreport.Clients) *
			cfg.GetInt("clientsPerMachine", 0) *
			cfg.GetInt("clientsNumberOfCommands", 0) *
			cfg.GetInt("clientsNumberOfRuns", 0)
		return float64(decidedRequestsCounter) /
			float64(expectedDecidedRequests) *
			100
	}

	return 0
}
