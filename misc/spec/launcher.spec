
Specification for the Launcher

While these design choices might be seem obvious and reasonable, it took us a
long time and much trial and error to reach this design.

(Check out the Pigeon paper for similar formulations.)

We draw inspiration from numerous sources:
- Pigeon and Falcon
- launchd: local process restarter
- http://launchd.info/
- http://supervisord.org/
- systemd, Upstart??
- gmx (davecheny)
- runsit (bradfitz)

Requirements and Features:
- Must be a generic process launcher (not tied to Goxos)

- Launcher process should be resident always to monitor that
  subprocesses have not crashed.

- Launcher runs only locally, but remote clients can connect over ssh or
  mosh or other means of remote login. This could be a custom solution with
  Websockets or TLS.

- Embed ssh client in 'launch' command to launch itself on machines in
  configuration.

- These subcommands should be invoked from the command line, or using APIs:
  - Requests to launch a new subprocess.
  - Query the health of existing subprocesses.
  - Subscribe to failure notifications.
  - The query 'list' command should be able to give results per host and per
    service (filtered by label); should support filtering by glob pattern:
      launch list <hostnames>
      launch list <service label>


- If launcher crashes, it will loose connection to all its
  subprocesses. So it cannot monitor them anymore.
  - How to deal with that?
  - New launcher: 
    - looks in persisent storage for pid of subprocesses
    - if any of those pid's remain.

--- MAYBE its not worth it. Just let them die with the launcher, as if the
--- host died. Then we don't need to use separate process groups. Then the
--- launcher can just restart; if any old pids remains, kill them off.

*** An alternative could be to leave them until a replacement has been
*** installed to avoid the reduced redundancy, but we will want to kill them
*** off before reusing the node for other replicas (with a new launcher
*** instance.)

- On restart of launch: either restart old processes (config option) or leave
  it be.

- The launcher may need to use a pid file to determine if it is still
  running?? Maybe we can connect over the domain socket to ask the launcher
  what its pid is. Meaning that if it responds, we know it is running at that
  pid.

- An extra design: the clients can help detect the crash of the launcher. By
  child processes (of the launcher) calling back over the domain socket or
  using the stdio pipes between them. This requires that the client volunteer
  to monitor the local launcher. If it cannot find its local launcher, it can
  commit suicide or report directly to the failure detection service over UDP
  multicast. Or do other application specific actions.


- Failure detectors requires the launchers to authenticate themselves;
  otherwise anyone can connect and push failure notifications to the failure
  detectors. We can use WebSocket with https.

/*

Our failure detection service is interested in failure notifications from
'launcher' entities deployed at all nodes. But since the launcher could also
fail, or the network in-between could fail, we cannot rely exclusively on
these failure notifications. Thus, the fallback will always be a timeout at
the application level. This will also capture the rare loss of a failure
notification due to message loss, inherent to UDP multicast.

This means that each launcher entity can simply multicast failure
notifications to a specific address to which interested failure detection
service entities can listen. This will work nicely since each data center will
have its own failure detection service entity and we assume that the data
center is configured to allow multicast within the data center.

Failure notifications sent over multicast must be authenticated as coming from
a real launcher.

Make test cases: - Start multiple launchers locally, and corresponding
processes and kill these processes in various ways and observe that the
failure detection service sees the notifications (unless there is network
problems or message loss etc.)

*/

Launcher:
- Receive request to launch a process on local Host.
- stdout/err should be redirected to a file

- Design alternatives:
  - ssh server (check out fab)
  	- Pro: no need for separate client
  	- Con: process must want to be monitored 
           (must register with ProcessSensor)
  	- Q: can we get feedback on whether a service started?
  	  A: Must query the Host sensor to see that it is in
  	  the process table.
  - RPC server with Command.Exec()
  	- Pro: can signal/kill subprocesses easily
  	- Pro: process does not need to tell sensor that it
  		   wishes to be monitored.
  	- Con: need custom client to launch processes from
  	- Con: not compatible with fab experiment files.

- Can serve as the host for ProcessSensor.
  - If so, ProcessSensor can use the Wait() call to detect
    process crashes/exits.

