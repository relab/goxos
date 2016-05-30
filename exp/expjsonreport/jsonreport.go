package expjsonreport

type JSONReport struct {
	ExperimentName              string
	TotalAverageThroughput      float64
	NumberOfValidRuns           int
	TotalNumberOfRuns           int
	Timestamp                   string
	PercentageOfRequestsDecided float64
	ExperimentFolderName        string
}
