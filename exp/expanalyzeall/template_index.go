package main

const templateIndex = `
<html>

<head>
<meta charset="utf-8">
<title>Experiment reports</title>

<link href="http://netdna.bootstrapcdn.com/bootswatch/3.0.2/spacelab/bootstrap.min.css" rel="stylesheet">

<head>

<body>

	<div class="container">
		<div class="page-header">
			<h1>Experiment reports</h1>
		</div>

		<ul class="nav nav-tabs" data-tabs="tabs">
		    <li class="active"><a data-toggle="tab" href="#all">All</a></li>
		    {{range .ReportsByExperimentName}}
		    <li><a data-toggle="tab" href="#{{.ExperimentName}}">{{.ExperimentName}}</a></li>
		    {{end}}
		</ul>
		<div class="tab-content">
			{{range .ReportsByExperimentName}}
		    <div class="tab-pane" id="{{.ExperimentName}}">
		        <h1>{{.ExperimentName}}</h1>

		        <div class="panel panel-default">
					<div class="panel-heading">
						<h3 class="panel-title">Average throughputs</h3>
					</div>
					<div class="panel-body">
						<a href="{{.ThroughputPlot}}">
							<img src="{{.ThroughputPlot}}" height="250px" width="100%" />
						</a>
					</div>	
				</div>

				<div class="panel panel-default">
					<div class="panel-heading">
						<h3 class="panel-title">Percentages of completed requests</h3>
					</div>
					<div class="panel-body">
						<a href="{{.RequestsCompletedPlot}}">
							<img src="{{.RequestsCompletedPlot}}" height="250px" width="100%" />
						</a>
					</div>	
				</div>

		        <div class="list-group">
					{{range .ExperimentReports}}
					

					<div class="row">
						<a href="Reports/{{.ExperimentFolderName}}/index.html" class="list-group-item">
							<h4 class="list-group-item-heading">
								<span class="span6">{{.ExperimentName}} - {{.Timestamp}}</span>
								<!--<span class="span1 pull-right">{{.TotalAverageThroughput}} req/sec</span>-->
								<span class="badge pull-right">{{.NumberOfValidRuns}} of {{.TotalNumberOfRuns}} runs completed</span>
								<span class="badge pull-right">{{.TotalAverageThroughput}} req/sec</span>

							</h4>
						</a>
					</div>
					{{end}}
				</div>			
		    </div>
		    {{end}}

		    <div class="tab-pane active" id="all">
		        <h1>All</h1>
		        <div class="list-group">
					{{range .AllReports}}
					<div class="row">
						<a href="Reports/{{.ExperimentFolderName}}/index.html" class="list-group-item">
							<h4 class="list-group-item-heading">
								<span class="span6">{{.ExperimentName}} - {{.Timestamp}}</span>
								<!--<span class="span1 pull-right">{{.TotalAverageThroughput}} req/sec</span>-->
								<span class="badge pull-right">{{.NumberOfValidRuns}} of {{.TotalNumberOfRuns}} runs completed</span>
								<span class="badge pull-right">{{.TotalAverageThroughput}} req/sec</span>

							</h4>
						</a>
					</div>
					{{end}}
				</div>		
		    </div>
		</div>








	</div>
</body>

<script type="text/javascript" src="https://ajax.googleapis.com/ajax/libs/jquery/1.7.2/jquery.min.js"></script>

<script type="text/javascript" src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>

</html>`
