This folder contains the launcher for remote processes.

To use it, you first need to start the launcher in serve mode as follows:

 `launch serve`

This will accept connections from a local process (over Unix domain sockets),
and will allow starting new processes, stop processes already running, list
running processes, and shutdown the launcher and all processes running through
the launcher.

Note that we will want start a launch process on all machines in the cluster
to be able to start new processes (i.e. replicas) on all machines.

Now, the fact that we only accept connections from local processes is just a
convenience to ensure proper secure access to the machine to start processes.
It also will only allow a specific user to start processes through the
launcher, because the user must be the same user as the one starting the
launcher.

However, to allow remote processes to start new processes on our node, we must
use ssh.

This can be done through:

 `ssh <host_running_launcher> launch submit -l <label> -p <command name>`.

(I will implement a simple wrapper for this later, but for now, I've only been
testing the localhost version of this.)

However, I have implemented a ssh client for a slightly different purpose
already so the proof of concept is more or less there.

The following command can be started in a separate terminal. The purpose of
this command is to start listening for failure notifications of processes
running on machines with a launcher.

 `launch subscribe --serve`

The launcher will send notifications to processes that are subscribers running
in serve mode.

The following commands can be executed on the same machine as a launcher (in
serve mode). 

```
 launch submit -l p1 -p sleep 120
 launch submit -l p2 -p sleep 120
 launch submit -l p3 -p sleep 120
 launch submit -l p4 -p sleep 120
 launch submit -l o1 -p sleep 120
 launch submit -l o2 -p sleep 120
 launch submit -l o3 -p sleep 120
 launch submit -l o4 -p sleep 120
 launch list
 launch stop p1 p2
 launch stop p3
 launch stop o1 o2
 launch stop o3
 launch stop o4
 launch shutdown
```

Note that above, I've used really short labels, to simplify the command line
details. In practice, when we use a JSON configuration file or other format,
such as plist (XML), to provide programs to start on the launcher servers, we
should provide labels that look something like:

 `label: "github.com/relab/goxos/kvs/kvsd"`

These two does not need to run on the same machine as the launcher:

```
 launch subscribe --serve
 launch subscribe --serve -a :9001 -f 'o*'
```

The first command will receive all notifications for all failures (e.g. due to
a stop), while the second command will only receive events that match the
pattern provided, i.e. the ones with labels that start with the letter 'o'.
The subscribers will send their subscription to the launcher over a ssh
connection, but the launcher connects to each of the subscribers to send
notifications. (TODO: We might want to send notifications over reliable UDP
instead of TCP to avoid issues of failure/reconnect.)


