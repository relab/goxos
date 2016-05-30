
Specification for Failure Detector Services

Requirements:
- Reliable Failure Detection

Concepts:
- Host (machine or virtual machine)
- Switch (physical switch or open vswitch)
- Process
- Launcher
- Sensor (Spy, Snitch, Monitor)


Multiple processes may run on a Host.

A machine may have multiple Hosts in the form of VMs.
If so, each of VM will have its own Launcher and Sensors.


ProcessSensor:
- Listen for connections from processes that wish to be monitored.
  - Register(process)
  - Keep connection to 'process'
  - On connection to 'process' lost
  	- If 'process' in OS process table
  	  - Wait for a period of time
  	  - Continue to check OS process table
  	  - Ping 'process' for health status?? Try reconnect??
  	- Else: Report failure of 'process' to subscribers

- Accept Subscribe() calls for a given 'process' type, e.g. "kvs".
- Issue FailureEvents to subscribers.

- How to combine ProcessSensor with ProcessWaiting?


OSSensor:
- Use kill signals etc to detect reboot/restart/shutdown events
  - Report FailureEvent to subscribers. 


Launcher implementation:
- Implement


Tests:
- Large number of processes connect to the ProcessSensor
- Take down the network interface
- Kill process in various ways
