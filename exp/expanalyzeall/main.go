package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/relab/goxos/exp/expjsonreport"

	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plot"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plotter"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/plotutil"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/plotinum/vg"
)

const (
	throughputPlotFile          = "throughput.png"
	percentageCompletedPlotFile = "percentageCompletedGraph.png"
)

type Reports struct {
	AllReports              []*expjsonreport.JSONReport
	ReportsByExperimentName []NamedReports
}

type NamedReports struct {
	ExperimentReports     []*expjsonreport.JSONReport
	ExperimentName        string
	ThroughputPlot        string
	RequestsCompletedPlot string
}

const usage = `Usage:

	expanalyseall -exppath <path-to-experiments-folder> -reportpath <path-to-reports-output-folder> -htmlpath=<path-to-html-folder> -expanalyze-bin-path=<path-to-expanalyze-bin-folder>

example:
expanalyseall -exppath=ExperimentsFolder -reportpath=public_html/Reports -htmlpath=public_html -expanalyze-bin-path=bins/expanalyze
`

var (
	experimentFolderPath = flag.String("exppath", "", "Path to root experiments folder")
	reportFolderPath     = flag.String("reportpath", "", "Path to report output folder")
	htmlReportFolderPath = flag.String("htmlpath", "", "Path to HTML report output folder")
	expanalyzeBinaryPath = flag.String("expanalyze-bin-path", "", "Path to expanalyze binary")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage, "\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if *experimentFolderPath == "" ||
		*reportFolderPath == "" ||
		*htmlReportFolderPath == "" ||
		*expanalyzeBinaryPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	checkImageAndReportsFolder()

	experimentPathsWithoutReports := findExperimentsWithoutReports()
	fmt.Println("Experiments found without report: ", len(experimentPathsWithoutReports))

	for _, experimentPath := range experimentPathsWithoutReports {
		analyzeExperiment(experimentPath)
	}

	generateHTMLReportsPage()
}

func checkImageAndReportsFolder() {
	err := os.MkdirAll(path.Join(*htmlReportFolderPath, "images"), 0755)
	if err != nil {
		fmt.Println("Error generating images folder: ", err)
	}
	err = os.MkdirAll(*reportFolderPath, 0755)
	if err != nil {
		fmt.Println("Error generating report folder path: ", err)
	}
}

func analyzeExperiment(experimentPath string) {
	fmt.Println("Analyzing: ", experimentPath)
	cmd := exec.Command(*expanalyzeBinaryPath, "-path="+experimentPath, "-rplot -clatana")
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting expanalyze: ", err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for expanalyze: ", err)
		return
	}
	moveReportToReportPath(experimentPath)
}

func moveReportToReportPath(experimentPath string) {
	pathParts := strings.Split(experimentPath, string(os.PathSeparator))
	experimentName := pathParts[len(pathParts)-1]
	err := os.Rename(path.Join(experimentPath, "report"), path.Join(*reportFolderPath, experimentName))
	if err != nil {
		fmt.Println("Error moving report: ", err)
	}
}

func findExperimentsWithoutReports() []string {
	fmt.Println("Finding experiments without reports")
	var pathsWithoutReports []string
	exppathFileInfos, err := getFileInfos(*experimentFolderPath)
	if err != nil {
		fmt.Println("Error when getting fileInfos exppath: ", err)
	}
	fmt.Println("Number of fileInfos in experiment folder: ", len(exppathFileInfos))
	reportpathFileInfos, err := getFileInfos(*reportFolderPath)
	if err != nil {
		fmt.Println("Error when getting fileInfos reportpath: ", err)
	}
	fmt.Println("Number of fileInfos in reportpath folder: ", len(reportpathFileInfos))

	for _, exppathFileInfo := range exppathFileInfos {
		if !isDirectoryInListOfDirectories(exppathFileInfo, reportpathFileInfos) {
			pathsWithoutReports = append(pathsWithoutReports, path.Join(*experimentFolderPath, exppathFileInfo.Name()))
		}
	}

	return pathsWithoutReports
}

func getFileInfos(directoryPath string) ([]os.FileInfo, error) {
	directory, err := os.Open(directoryPath)
	if err != nil {
		return nil, err
	}

	defer directory.Close()

	fileInfos, err := directory.Readdir(-1)
	if err != nil {
		return nil, err
	}
	return fileInfos, nil
}

func isDirectoryInListOfDirectories(dir os.FileInfo, directories []os.FileInfo) bool {
	for _, fileInfo := range directories {
		if dir.Name() == fileInfo.Name() {
			return true
		}
	}
	return false
}

func generateHTMLReportsPage() {
	jsonreports, err := getJSONReports()
	if err != nil {
		fmt.Println("Error getting JSON reports: ", err)
		return
	}
	reports := createReportsStructForHTML(jsonreports)
	bytesBuffer := new(bytes.Buffer)
	reportsTemplate := template.Must(template.New(templateIndex).Parse(templateIndex))
	err = reportsTemplate.Execute(bytesBuffer, reports)
	if err != nil {
		fmt.Println("Error executing reports template: ", err)
		return
	}

	err = ioutil.WriteFile(path.Join(*htmlReportFolderPath, "index.html"), bytesBuffer.Bytes(), 0644)
	if err != nil {
		fmt.Println("Error writing html to disk: ", err)
	}
}

func createReportsStructForHTML(jsonReports []*expjsonreport.JSONReport) Reports {
	reports := Reports{AllReports: jsonReports, ReportsByExperimentName: make([]NamedReports, 0)}
	//divide all reports into namedReports by experiment name
	for _, report := range jsonReports {

		index := -1
		for i, namedReports := range reports.ReportsByExperimentName {
			if namedReports.ExperimentName == report.ExperimentName {
				index = i
				reports.ReportsByExperimentName[i].ExperimentReports = append(reports.ReportsByExperimentName[i].ExperimentReports, report)
			}
		}
		//if no namedReports yet with that name, create new:
		if index == -1 {
			newNamedReports := NamedReports{ExperimentReports: make([]*expjsonreport.JSONReport, 0), ExperimentName: report.ExperimentName}
			newNamedReports.ExperimentReports = append(newNamedReports.ExperimentReports, report)
			reports.ReportsByExperimentName = append(reports.ReportsByExperimentName, newNamedReports)
		}
	}

	// generate plots for each experiment name/type
	for i, rep := range reports.ReportsByExperimentName {
		fmt.Println("Generating plot for: ", rep.ExperimentName)
		err := generateThroughputGraph(rep.ExperimentReports, path.Join(*htmlReportFolderPath, "images", rep.ExperimentReports[0].ExperimentName+throughputPlotFile))
		if err != nil {
			fmt.Println("Error generating throughput plot for experiment: ", err)
		} else {
			rep.ThroughputPlot = "images/" + rep.ExperimentReports[0].ExperimentName + throughputPlotFile
		}
		err = generatePercentageCompletedGraph(rep.ExperimentReports, path.Join(*htmlReportFolderPath, "images", rep.ExperimentReports[0].ExperimentName+percentageCompletedPlotFile))
		if err != nil {
			fmt.Println("Error generating percentage completed plot for experiment: ", err)
		} else {
			rep.RequestsCompletedPlot = path.Join("images", rep.ExperimentReports[0].ExperimentName+percentageCompletedPlotFile)
		}
		reports.ReportsByExperimentName[i] = rep
	}
	return reports
}

func getJSONReports() ([]*expjsonreport.JSONReport, error) {
	var reports []*expjsonreport.JSONReport

	reportsFileInfos, err := getFileInfos(*reportFolderPath)
	if err != nil {
		return nil, err
	}

	// for _, fileinfo := range reportsFileInfos {
	// 	report, err := getJSONReport(reportFolderPath + "/" + fileinfo.Name() + "/" + "jsonReport.json")
	// 	if err != nil {
	// 		fmt.Println("Error reading JSON file: ", err)
	// 		continue
	// 	}
	// 	reports = append(reports, report)
	// }

	//loop in reverse order to get newest first on webpage
	for i := len(reportsFileInfos) - 1; i >= 0; i-- {
		report, err := getJSONReport(path.Join(*reportFolderPath, reportsFileInfos[i].Name(), "jsonReport.json"))
		if err != nil {
			fmt.Println("Error reading JSON file: ", err)
			continue
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func getJSONReport(path string) (*expjsonreport.JSONReport, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report expjsonreport.JSONReport
	err = json.Unmarshal(bytes, &report)

	if err != nil {
		return nil, err
	}
	pathParts := strings.Split(path, string(os.PathSeparator))
	report.ExperimentFolderName = pathParts[len(pathParts)-2] // equals the folder name of the report

	//report.ExperimentName = report.ExperimentName + " - " + report.Timestamp

	return &report, nil
}

func generateThroughputGraph(jsonreports []*expjsonreport.JSONReport, savePath string) error {
	plot, err := plot.New()
	if err != nil {
		fmt.Println("Error creating new plot: ", err)
		return err
	}

	plot.Add(plotter.NewGrid())
	plot.Title.Text = "Average replica throughput"
	plot.X.Label.Text = "Experiment"
	plot.Y.Label.Text = "Average throughput"

	points := make(plotter.XYs, len(jsonreports))

	// for i, report := range jsonreports {
	// 	points[i].X = float64(i) // TODO: maybe something else here? date perhaps?
	// 	points[i].Y = report.TotalAverageThroughput
	// }
	xCounter := 0
	for i := len(jsonreports) - 1; i >= 0; i-- {
		points[i].X = float64(xCounter) // TODO: maybe something else here? date perhaps?
		points[i].Y = jsonreports[i].TotalAverageThroughput
		xCounter++
	}

	line, err := plotter.NewLine(points)
	if err != nil {
		fmt.Println("Error creating line in plot: ", err)
		return err
	}

	line.LineStyle.Width = vg.Points(5)
	//line.LineStyle.Dashes = plotutil.Dashes(1)
	line.LineStyle.Color = plotutil.Color(1)
	plot.Add(line)

	//plot.Save(20, 11, htmlReportPath+throughputPlotFile)
	plot.Save(20, 11, savePath)
	return nil
}

func generatePercentageCompletedGraph(jsonreports []*expjsonreport.JSONReport, savePath string) error {
	plot, err := plot.New()
	if err != nil {
		fmt.Println("Error creating new plot: ", err)
		return err
	}

	plot.Add(plotter.NewGrid())
	plot.Title.Text = "Percentage of requests decided"
	plot.X.Label.Text = "Experiment"
	plot.Y.Label.Text = "Percentage"

	points := make(plotter.XYs, len(jsonreports))

	// for i, report := range jsonreports {
	// 	points[i].X = float64(i) // TODO: maybe something else here? date perhaps?
	// 	points[i].Y = report.PercentageOfRequestsDecided
	// }
	xCounter := 0
	for i := len(jsonreports) - 1; i >= 0; i-- {
		points[i].X = float64(xCounter) // TODO: maybe something else here? date perhaps?
		points[i].Y = jsonreports[i].PercentageOfRequestsDecided
		xCounter++
	}

	line, err := plotter.NewLine(points)
	if err != nil {
		fmt.Println("Error creating line in plot: ", err)
		return err
	}

	line.LineStyle.Width = vg.Points(5)
	//line.LineStyle.Dashes = plotutil.Dashes(1)
	line.LineStyle.Color = plotutil.Color(1)
	plot.Add(line)

	plot.Save(20, 11, savePath)
	return nil
}
