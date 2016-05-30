package main

const templateRun = `
<html>

<head>
<meta charset="utf-8">
<title>Run {{.ID}}</title>

<link href="http://netdna.bootstrapcdn.com/bootswatch/3.0.2/spacelab/bootstrap.min.css" rel="stylesheet">

<head>

<body>

<div class="container">

	<div class="page-header">
		{{if .Valid}}
		<h1> 
			Run {{.ID}}
			<span class="label label-success">Valid</span>
		</h1>
		{{else}}
		<h1> 
			Run {{.ID}}
			<span class="label label-danger">Invalid</span>
		</h1>
		{{end}}

	</div>

	{{if .Errors}}
	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Errors</h3>
		</div>
		<div class="panel-body">
			<ul>
				{{range .Errors}}
				<li class="alert alert-danger">{{.}}</li>
				{{end}}
			</ul>
		</div>
	</div>
	{{end}}

	{{if .Warnings}}
	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Warnings</h3>
		</div>
		<div class="panel-body">
			<ul>
				{{range .Warnings}}
				<li class="alert alert-warning">{{.}}</li>
				{{end}}
			</ul>
		</div>
	</div>
	{{end}}

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">State hashes</h3>
		</div>
		<div class="panel-body">
			{{if .StateHashes }}
			<ul>
				{{range $replica, $hash := .StateHashes}}
				<li><b>{{$replica}}</b>: <pre>{{$hash}}</pre></li>
				{{end}}
			</ul>
			{{else}}
			<p>No state hashes found...</p>
			{{end}}
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Timeline</h3>
		</div>
		<div class="panel-body">
			{{if .Timeline}}
			<ol>
				{{range .Timeline}}
				<li><b>{{.Replica}}:</b> {{.EventDesc}}</li>
				{{end}}
			</ol>
			{{else}}
			<p>No timeline found...</p>
			{{end}}
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Logs</h3>
		</div>
		<div class="panel-body">
			<div class="list-group">
				{{if .LogURLs}}
				<ul class="list-group">
					{{range $replica, $urls := .LogURLs}}
					<li class="list-group-item">
					<b>{{$replica}}</b>: 
					<a href="{{$urls.Elog}}">elog</a>,
					<a href="{{$urls.Glog}}">glog</a>
					and <a href="{{$urls.Stderr}}">stderr</a>	
					</li>
					{{end}}
				</ul>
				{{else}}
				<p>No replicas found...</p>
				{{end}}
			</div>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Durations</h3>
		</div>
		<div class="panel-body">
			{{if .Valid}}
			<table class="table">
				<thead>
					<tr>
						<th>Failure handling interval</th>
						<th>Duration</th>
					</tr>
				</thead>
				<tbody>
				<tr>
					<td>
						Initialization:
					</td>
					<td>
						{{.InitDur}}
					</td>
				</tr>
				<tr>
					<td>
						Activation:
					</td>
					<td>
						{{.ActiDur}}
					</td>
				</tr>
				<tr>
					<td>
						Wait for First Acc:
					</td>
					<td>
						{{.FirstAcc}}
					</td>
				</tr>
				{{if .ARecActiDur}}
				<tr>
					<td>
						Activated from CPromises:
					</td>
					<td>
						{{.ARecActiDur}}
					</td>
				</tr>
				{{end}}
				</tbody>
			</table>
			{{else}}
			<p>No durations calculated...</p>
			{{end}}		
		</div>
	</div>

	<div class="panel panel-default">
                <div class="panel-heading">
                        <h3 class="panel-title">ClientLatenciess</h3>
                </div>
                <div class="panel-body">
                        {{if .Valid}}
                        <table class="table">
                                <thead>
                                        <tr>
                                                <th>Client Latencies</th>
                                                <th>Latency in ms</th>
                                        </tr>
                                </thead>
                                <tbody>
                                <tr>
                                        <td>
                                                Median:
                                        </td>
                                        <td>
                                                {{.MedianClLatency}}
                                        </td>
                                </tr>
                                <tr>
                                        <td>
                                                Maximum:
                                        </td>
                                        <td>
                                                {{.MaxClLatency}}
                                        </td>
                                </tr>
				<tr>
					<td>
						Maximum StartTime:
					</td>
					<td>
						{{.MaxClLatencyTs}}
					</td>
				</tr>
				<tr>
					<td>
						Maximum Latency During Activation:
					</td>
					<td>
						{{.MaxActClLatency}}
					</td>
				</tr>
				<tr>
					<td>
						Maximum Activation Latency StartTime:
					</td>
					<td>
						{{.MaxActClLatencyTs}}
					</td>
				</tr>
                                </tbody>
                        </table>
                        {{else}}
                        <p>No latencies calculated...</p>
                        {{end}}         
                </div>
        </div>
	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Throughput</h3>
		</div>
		<div class="panel-body">
			{{if .Valid}}
			<table class="table">
			<thead>
					<tr>
						<th>Average</th>
						<th>Paxos leader throughput (req/sec)</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td>
						Average throughtput :
						</td>
						<td>
						{{.AverageThroughput}}
						</td>
					</tr>
				</tbody>
				<thead>
					<tr>
						<th>Failure handling interval</th>
						<th>Paxos leader throughput (req/sec)</th>
					</tr>
				</thead>
				<tbody>
				<tr>
					<td>
						Baseline mean :
					</td>
					<td>
						{{.BaseTputMean}}
					</td>
				</tr>
				<tr>
					<td>
						Baseline (standard deviation):
					</td>
					<td>
						{{.BaseTputSSD}}
					</td>
				</tr>
				<tr>
					<td>
						Base (standard error of the mean):
					</td>
					<td>
						{{.BaseTputStdErrOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Baseline (min):
					</td>
					<td>
						{{.BaseTputMin}}
					</td>
				</tr>
				<tr>
					<td>
						Base (max):
					</td>
					<td>
						{{.BaseTputMax}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (mean):
					</td>
					<td>
						{{.InitTputMean}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (standard deviation):
					</td>
					<td>
						{{.InitTputSSD}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (standard error of the mean):
					</td>
					<td>
						{{.InitTputStdErrOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (min):
					</td>
					<td>
						{{.InitTputMin}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (max):
					</td>
					<td>
						{{.InitTputMax}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (mean):
					</td>
					<td>
						{{.ActiTputMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (standard deviation):
					</td>
					<td>
						{{.ActiTputSSD}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (standard error of the mean):
					</td>
					<td>
						{{.ActiTputStdErrOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (min):
					</td>
					<td>
						{{.ActiTputMin}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (max):
					</td>
					<td>
						{{.ActiTputMax}}
					</td>
				</tr>
				</tbody>
			</table>
			{{else}}
			<p>No throughput statistics calculated...</p>
			{{end}}		
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Replica throughput plots</h3>
		</div>
		<div class="panel-body">
			<div class="list-group">
				{{if .ThroughputPlot}}
				<li class="list-group-item">
				<a href="{{.ThroughputPlot}}">
					<img src="{{.ThroughputPlot}}" height="200px" width="100%" />
				</a>
				</li>
				{{else}}
				<p>No plots generated...</p>
				{{end}}
			</div>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Client latency plots</h3>
		</div>
		<div class="panel-body">
			<div class="list-group">
				{{if .ClientPlots}}
				{{range $client, $imgsrc := .ClientPlots }}
				<li class="list-group-item">
				<h4 class="list-group-item-heading">
					{{$client}}
				</h4>
				<a href="{{$imgsrc}}">
					<img src="{{$imgsrc}}" height="200px" width="100%" />
				</a>
				</li>
				{{end}}
				{{else}}
				<p>No plots generated...</p>
				{{end}}
			</div>
		</div>
	</div>

</div>

</body>
</html>`
