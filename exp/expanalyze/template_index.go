package main

const templateIndex = `
<html>

<head>
<meta charset="utf-8">
<title>{{.Cfg.GetString "name" "Unnamed"}}</title>

<link href="http://netdna.bootstrapcdn.com/bootswatch/3.0.2/spacelab/bootstrap.min.css" rel="stylesheet">

<head>

<body>

<div class="container">

	<div class="page-header">
		<h1>Experiment report: {{.Cfg.GetString "name" "Unnamed"}}</h1>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">General</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>Date: {{.Timestamp}}</li>
				<li>Duration: {{.Cfg.GetString "duration" "N/A"}}</li>
				<li>Runs: {{.Cfg.GetInt "runs" 0}}</li>
				<li>Remote user: {{.Cfg.GetString "remoteUser" "N/A"}}</li>
				<li>Remote scratch dir: {{.Cfg.GetString "remoteScratchDir" ""}}</li>
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Versions</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>{{.Go}}</li>
				<li>goxos: {{.Goxos}}</li>
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Cluster</h3>
		</div>
		<div class="panel-body">
			<p>Replicas:</p>
			<ul>
				{{range .Hostnames.Nodes}}
				<li>{{.}}</li>
				{{else}}					
				<li>None found</li>
				{{end}}					
			</ul>
			<p>Standbys:</p>
			<ul>
				{{range .Hostnames.Standbys}}
				<li>{{.}}</li>
				{{else}}					
				<li>None found</li>
				{{end}}					
			</ul>
			<p>Clients:</p>
			<ul>
				{{range .Hostnames.Clients}}
				<li>{{.}}</li>
				{{else}}					
				<li>None found</li>
				{{end}}					
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Failures</h3>
		</div>
		<div class="panel-body">
			{{if .Hostnames.Failures}}
<!--			<ul>
				{{range .Hostnames.Failures}}
				<li>{{.}} (+{{/* .Time */}} seconds)</li>
				{{end}}
			</ul> -->
      <ul>{{.Cfg.GetString "failures" ""}}</ul>
			{{else}}
			<p>No failures defined for this experiment.</p>
			{{end}}
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">State</h3>
		</div>
		<div class="panel-body">
			<ul>
				{{if (.Cfg.GetString "replicaInitialState" "")}}
				<li>Initial replica state: {{.Cfg.GetString "replicaInitialState" ""}}</li>
				{{else}}
				<li>Initial replica state: None</li>	
				{{end}}
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Paxos Settings</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>Paxos type: {{.Cfg.GetString "protocol" ""}}</li>
				<li>Alpha: {{.Cfg.GetInt "alpha" 0}}</li>
				<li>BatchMaxSize: {{.Cfg.GetInt "batchMaxSize" 0}}</li>
				<li>BatchTimeout: {{.Cfg.GetString "batchTimeout" "N/A"}}</li>
				<li>Failure handling type: {{.Cfg.GetString "failureHandlingType" ""}}</li>
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Liveness Settings</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>Heartbeat emitter interval: {{.Cfg.GetString "hbEmitterInterval" "N/A"}}</li>
				<li>Failure detector timeout interval: {{.Cfg.GetString "fdTimeoutInterval" "N/A"}}</li>
				<li>Failure detector delta increase: {{.Cfg.GetString "fdDeltaIncrease" "N/A"}}</li>
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Client Settings</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>Clients per machine: {{.Cfg.GetString "clientsPerMachine" "N/A"}}</li>
				<li>Number of runs: {{.Cfg.GetString "clientsNumberOfRuns" "N/A"}}</li>
				<li>Number of commands: {{.Cfg.GetString "clientsNumberOfCommands" "N/A"}}</li>
				<li>Key size: {{.Cfg.GetString "clientsKeySize" "N/A"}} bytes</li>
				<li>Value size: {{.Cfg.GetString "clientsValueSize" "N/A"}} bytes</li>
			</ul>
		</div>
	</div>

	{{if .LREnabled}}
	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Live Replacement Settings</h3>
		</div>
		<div class="panel-body">
			<ul>
				{{if (.Cfg.GetString "LRExpRndSleep" "")}}
				<li>Random pre-connecy sleep enabled: {{.Cfg.GetString "LRExpRndSleep" ""}} ms</li>
				{{else}}
				<li>Random pre-connect sleep not enabled</li>
				{{end}}
				<li>Strategy: {{.Cfg.GetString "LRStrategy" "N/A"}}</li>
			</ul>
		</div>
	</div>
	{{end}}

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Logging</h3>
		</div>
		<div class="panel-body">
			<ul>
				<li>glog v-level: {{.Cfg.GetString "glogVLevel" "N/A"}}</li>
				{{if (.Cfg.GetString "glogVModule" "")}}
				<li>glog vmodule argument: {{.Cfg.GetString "glogVModule" "N/A"}}</li>
				{{end}}
				<li>Throughput sampling interval: {{.Cfg.GetString "throughputSamplingInterval" "N/A"}}</li>
			</ul>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Experiment Runs ({{.ValidRuns}} of {{.Cfg.GetString "runs" "0"}} valid)</h3>
		</div>
		<div class="panel-body">
			<div class="list-group">
				{{range .ExpRuns}}
				<a href="{{.ID}}/{{.ID}}.html" class="list-group-item">
					{{if .Valid}}
					<h4 class="list-group-item-heading">
						Run {{.ID}}
						<span class="label label-success">Valid</span>
					</h4>
					{{else}}
					<h4 class="list-group-item-heading">
						Run {{.ID}}
						<span class="label label-danger">Invalid</span>
					</h4>
					{{end}}
				</a>
				{{else}}
				<a href="#" class="list-group-item">
					<h4 class="list-group-item-heading">No runs found...</h4>
				</a>
				{{end}}
			</div>
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Durations</h3>
		</div>
		<div class="panel-body">
			{{if .Analysed}}
			<table class="table">
				<thead>
					<tr>
						<th>Failure handling interval</th>
						<th>Duration</th>
					</tr>
				</thead>
				<tbody>
				<tr class="success">
					<td>
						Initialization (mean):
					</td>
					<td>
						{{.InitDurMean}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (standard deviation):
					</td>
					<td>
						{{.InitDurSD}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (standard error of mean):
					</td>
					<td>
						{{.InitDurSdOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (min):
					</td>
					<td>
						{{.InitDurMin}}
					</td>
				</tr>
				<tr>
					<td>
						Initialization (max):
					</td>
					<td>
						{{.InitDurMax}}
					</td>
				</tr>
				<tr class="success">
					<td>
						Activation (mean):
					</td>
					<td>
						{{.ActiDurMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (standard deviation):
					</td>
					<td>
						{{.ActiDurSD}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (standard error of mean):
					</td>
					<td>
						{{.ActiDurSdOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (min):
					</td>
					<td>
						{{.ActiDurMin}}
					</td>
				</tr>
				<tr>
					<td>
						Activation (max):
					</td>
					<td>
						{{.ActiDurMax}}
					</td>
				</tr>
				<tr class="success">
					<td>
						Wait for Acc (mean):
					</td>
					<td>
						{{.FAccDurMean}}
					</td>
				</tr>
				<tr>
					<td>
						First Acc (standard deviation):
					</td>
					<td>
						{{.FAccDurSD}}
					</td>
				</tr>
				<tr>
					<td>
						First Acc (standard error of mean):
					</td>
					<td>
						{{.FAccDurSdOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						First Acc (min):
					</td>
					<td>
						{{.FAccDurMin}}
					</td>
				</tr>
				<tr>
					<td>
						First Acc (max):
					</td>
					<td>
						{{.FAccDurMax}}
					</td>
				</tr>

				{{if .ARecEnabled}}
				<tr class="success">
					<td>
						Activated from CPromises (mean):
					</td>
					<td>
						{{.ARecActiDurMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activated from CPromises (standard deviation):
					</td>
					<td>
						{{.ARecActiDurSD}}
					</td>
				</tr>
				<tr>
					<td>
						Activated from CPromises (standard error of mean):
					</td>
					<td>
						{{.ARecActiDurSdOfMean}}
					</td>
				</tr>
				<tr>
					<td>
						Activated from CPromises (min):
					</td>
					<td>
						{{.ARecActiDurMin}}
					</td>
				</tr>
				<tr>
					<td>
						Activated from CPromises (max):
					</td>
					<td>
						{{.ARecActiDurMax}}
					</td>
				</tr>
				{{end}}
				</tbody>
			</table>
			{{else}}
			<div class="alert alert-danger">{{.Comment}}</div>
			{{end}}
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Throughput</h3>
		</div>
		<div class="panel-body">
			<table class="table">
				<thead>
					<tr>
						<th>Average throughput</th>
						<th>Paxos leader throughput (req/sec)</th>
					</tr>
				</thead>
				<tbody>
					<tr>
					<td>
						Total average throughput:
					</td>
					<td>
						{{.TotalAverageThroughput}}
					</td>
				</tr>
				</tbody>
			</table>
		</div>
		<div class="panel-body">
			{{if .Analysed}}
			<table class="table">
				<thead>
					<tr>
						<th>Failure handling interval</th>
						<th>Paxos leader throughput (req/sec)</th>
					</tr>
				</thead>
				<tbody>
				<tr class="success">
					<td>
						Baseline (mean):
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
						Baseline (standard error of mean):
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
						Baseline (max):
					</td>
					<td>
						{{.BaseTputMax}}
					</td>
				</tr>
				<tr class="success">
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
						Initialization (standard error of mean):
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
				<tr class="success">
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
						Activation (standard error of mean):
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
			<div class="alert alert-danger">{{.Comment}}</div>
			{{end}}
		</div>
	</div>

	<div class="panel panel-default">
		<div class="panel-heading">
			<h3 class="panel-title">Latencies</h3>
		</div>
		<div class="panel-body">
			{{if .Analysed}}
			<table class="table">
				<thead>
					<tr>
						<th>Latencies</th>
						<th>in ms</th>
					</tr>
				</thead>
				<tbody>
				<tr class="success">
					<td>
						Median Latency :
					</td>
					<td>
						{{.MeanLatency}}
					</td>
				</tr>
				<tr>
					<td>
						Median 90% Low Error:
					</td>
					<td>
						{{.MeanLatMin90}}
					</td>
				</tr>
				<tr>
					<td>
						Median 90% High Error:
					</td>
					<td>
						{{.MeanLatMax90}}
					</td>
				</tr>

				<tr>
					<td>
						Spike Latency (Median):
					</td>
					<td>
						{{.MeanMaxLatency}}
					</td>
				</tr>
				<tr>
					<td>
						Activation Latency (Median):
					</td>
					<td>
						{{.MeanMaxActLatency}}
					</td>
				</tr>
				<tr>
					<td>
						Activation Latency Low 90% Error:
					</td>
					<td>
						{{.MMALMin90}}
					</td>
				</tr>
				<tr>
					<td>
						Activation Latency High 90% Error:
					</td>
					<td>
						{{.MMALMax90}}
					</td>
				</tr>

				</tbody>
			</table>
			{{else}}
			<div class="alert alert-danger">{{.Comment}}</div>
			{{end}}
		</div>
	</div>

</div>

</body>

</html>`
