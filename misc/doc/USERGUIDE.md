% Goxos User's Guide
% Stephen Michael Jothen; Tormod Erevik Lea
% December 26, 2013

Goxos is a framework for building a replicated service. This guide will
introduce the user to building replicated services using the Goxos
framework. First, we will go over the basic concepts, and then we will
dive into a real-world example explaining how to build a replicated
key-value store.

Introduction
============

The first key to understanding how to build a fault-tolerant replicated
service using Goxos is to look at the interface available to developers.

Creating a replicated service can be done with the
`goxos.NewGoxosReplica` function. This function will construct a new
replica, and has the following signature:

    func NewGoxosReplica(uint, string, string, app.Handler) *Goxos

The first two arguments to `NewGoxosReplica` are the id of the replica,
and the id of the application. Next is a string denoting the path to the
configuration file. Finally, the fourth argument is a type that
implements the `app.Handler` interface.

The `app.Handler` interface is what an application that is to be
replicated must implement. This interface defines a few methods that
must be implemented on the type. These methods are:

    type Handler interface {
      Execute(req []byte) (resp []byte)
      GetState(slotMarker uint) (sm uint, state []byte)
      SetState(state []byte) error
    }

The first method, `Execute` takes a byte slice, which should be some
sort of command that can be executed in the application. The `Execute`
method also returns a response from the application in the form of a
byte slice.

The second and third methods, `GetState` and `SetState` are used for
live replacement, but could also be used for snapshotting in the future.

This is one way to look at the replicated service, from the point of
view of the server. However, it is also useful to look at it from the
other side, from the point of view of the client. The client library for
Goxos, located in the `goxosc` package, is used for connecting to the
replicated as well as sending and receiving responses. The client
connection can be created with the `Dial` method in the `goxosc`
package:

    func Dial() (*Conn, error)

This method returns a `Conn` representing a connection to the whole
replicated service. All of the dirty work of handshaking with the
servers and identifying the leader are abstracted away. The most useful
method on a `Conn` is `SendRequest`, which can be used to send requests
to the service:

    func (c *Conn) SendRequest(req []byte) ([]byte, error)

Some things to note about `SendRequest` is that the request is a byte
slice. This means that if you want to send Go structs or other complex
types as commands to the service, you must marshal them into byte form.
Another thing is that the return value is also a byte slice, and this
represents the response from the service. This also means that the
client-server interaction is synchronous, i.e. a client must wait for a
response before it can send any further requests.

Replicate Your Service -- Server Side
=====================================

We can now begin to discuss how to create an actual replicated service.
For this example, we will be giving an overview of how to create a
fault-tolerant key-value store. A key-value store is a simple type of
database with three basic operations:

-   `READ(key)` which returns the value associated with `key`.
-   `WRITE(key, value)` which associates `key` with `value`.
-   `DELETE(key)` which removes `key` from the store.

For simplicity's sake, we will assume that both `key` and `value` are of
type string. This means we can model the key-value store as a
`map[string]string`, which simplifies things for us greatly. This is
where we will discuss the main type which implements the `app.Handler`
interface.

    type GoxosHandler struct {
      smap   map[string]string
      buffer *bytes.Buffer
    }

As can be seen, the type `GoxosHandler` has some fields defined inside
of it. These fields make up the state of the replicated service, which
gets modified based on the incoming commands. Of course, to satisfy the
interface requirements, the methods described in the previous section
must be implemented for this type. First, however we will delve into the
details of how the commands are implemented. The first type is a
`MapRequest` which is a request to the key-value store to either read or
write a value:

    type CommandType int 

    type MapRequest struct {
      Ct    CommandType
      Key   string
      Value string
    }

The `MapRequest` struct contains three fields, the first one (`Ct`)
tells us what type of command this is (whether it is a `READ`, `WRITE`,
or `DELETE`). The second and third fields give us information about what
key is to be read or written to, as well as the value to write in the
latter case. The `CommandType` is just a type alias to `int` and can be
one of three predefined constants:

    const (
      READ = iota
      WRITE
      DELETE
    )

The second type we need to discuss is the `MapResponse` type, which the
client will receive as a response to one of its requests:

    type MapResponse struct {
      ToType CommandType
      Value  string
      Found  bool
      Err    string
    }

This type contains a `CommandType` which is the same as the field
contained in the corresponding `MapRequest`. It also contains fields
`Value` and `Found`, which contain the value in the case of a `READ` as
well as whether or not the entry was found in the key-value store.
`Found` could be false if trying to `READ` a key that doesn't exist.
Finally, the field `Err` is used to return errors back to the client.
With all of the discussion about the types used in the key-value store,
we can discuss the essential methods.

When a value is decided on a replica, it is passed to the `Execute`
method discussed in the previous section. This method is responsible for
applying meaning to the byte slice passed to it. Usually, as in our
case, this meaning is some form of a command. In our case, the commands
are structs, and these need to be unmarshalled from Go's `gob` format.
We can take a look at the `Execute` method to see how this works.

    func (gh *GoxosHandler) Execute(req []byte) (resp []byte) {
      var mreq kc.MapRequest
      var mresp kc.MapResponse

      gh.buffer.Reset()
      gh.buffer.Write(req)

      if err := gob.NewDecoder(gh.buffer).Decode(&mreq); err != nil {
        // Handle error in decoding ...
      }

      // Actual execution goes here ... which sets mresp.

      gh.buffer.Reset()
      gob.NewEncoder(gh.buffer).Encode(mresp)
      return gh.buffer.Bytes()
    }

In this code snippet, the execution and error handling code has been
left out; we can discuss these things later. The essential thing to note
is that the byte slice `req` is being written to an empty buffer, which
we then decode using `gob`. The `gob` decoder puts the decoded
`MapRequest` into the `mreq` variable. Finally, after the execution has
taken place, the buffer is reset and `mresp` is encoded from a
`MapResponse` to a byte slice, which is then returned. This return value
is sent back to the client by the `server` module.

The execution code, replaced with a comment in the previous code block,
is actually quite simple. All it needs to do is execute based on the
`CommandType` field in the `MapRequest` that was decoded:

    switch mreq.Ct {
    case kc.READ:
      val, ok := gh.smap[mreq.Key]
      mresp = kc.MapResponse{Value: val, Found: ok, ToType: kc.READ}
    case kc.WRITE:
      gh.smap[mreq.Key] = mreq.Value
      mresp = kc.MapResponse{Value: mreq.Value, ToType: kc.WRITE}
    case kc.DELETE:
      delete(gh.smap, mreq.Key)
      mresp = kc.MapResponse{ToType: kc.DELETE}
    default:
      mresp = kc.MapResponse{Err: "Unknown map command"}
    }

As can be seen in the above code block, what is done in the execution
phase is dependent on the type of command stored in the `Ct` field. In
case the case of `READ`, the value is read out of the map stored in the
`GoxosHandler` type, and a response is created. In the case of `WRITE`,
the key is associated to a value supplied in the `MapRequest`. The
`DELETE` case uses the built-in function `delete` to remove the mapping
for the key. Finally, if there is any other value stored in `Ct`, this
is a problem and this is where the `Err` field of the `MapResponse`
comes into play.

The `Err` field is also used in the error case omitted above. In this
case, the actual request cannot be unmarshalled from `gob` format and we
must signal an error back to the client:

    gh.buffer.Reset()
    mresp = kc.MapResponse{Found: false, Err: "Can't decode request"}
    gob.NewEncoder(gh.buffer).Encode(mresp)
    return gh.buffer.Bytes()

This section describes the server-side implementation, i.e. the
implementation of the application run by each of the replicas making up
the replicated service. However, the client-side is just as important.

Replicate Your Service -- Client Side
=====================================

The usage pattern of a client is dependent on what type of application
is using the replicated service. Due to this, we won't go into any
details about specific applications, but just explain how to use the
client interface in a general fashion.

The first thing that needs to be done by the client is to connect to the
replicated service. There's no way to issue requests without doing this
first! The client application must first connect to the replicated
service using the `Dial` function:

    g, err := goxosc.Dial()

The client interface, as discussed in the introduction section, is very
simple. For usage of the key-value store, all that needs to be done is
to create the desired `MapRequest` structure describing the type of
request you wish to send to the service. This then needs to be
marshalled to `gob` format so that it fits the requirements of the
`SendRequest` method:

    req = kc.MapRequest{kc.CommandType(ct - 1), key, value}

    if err := gob.NewEncoder(buffer).Encode(req); err != nil {
      // Handle error
    }   

In this case the marshalled request gets stored into a `Buffer`. Once
this is complete, the request can be taken out of the `Buffer` as a byte
slice and passed to the `SendRequest` method:

    byteResp, err := g.SendRequest(buffer.Bytes())

    if err != nil {
      // Handle error
    }   

Since `SendRequest` is synchronous, it also returns the response
received from the replicated service. However, it receives the response
as a byte slice, which needs to be unmarshalled into a `MapResponse`:

    buffer.Reset()
    buffer.Write(byteResp)
    var resp kc.MapResponse

    if err = gob.NewDecoder(buffer).Decode(&resp); err != nil {
      // Handle error
    }

Once this is complete, the response can be interpreted by the client
application, and further requests can be made to the replicated service.

Running
=======

So far, we've discussed how to create the replicated service, but
there's still one important part missing: How do we run the service and
the clients?

The application, described above, must first be compiled to a single
executable. This can be done simply using the `go build` functionality.
This executable should take the id of the replica as an argument. Once
this is done the replica can be booted up. For this example, we will
boot the replicas up on three different machines (`pitter`, `pitter2`,
and `pitter3`):

    pitter1$ go build
    pitter1$ ./kvs -id 0

When the replica is launched, it will also read the configuration file.
The id passed to the executable must correspond to one of the nodes
listed in the configuration file. This configuration file is a file
containing a JSON object. The object stores information about each of
the nodes, as well as different settings that can be tweaked. An example
of a configuration file is shown below:

    {
      "Nodes": {
        "0": {
          "Ip": "pitter1",
          "PaxosPort": "8080",
          "ClientPort": "8081"
        },
        "1": {
          "Ip": "pitter2",
          "PaxosPort": "8082",
          "ClientPort": "8083"
        },
        "2": {
          "Ip": "pitter3",
          "PaxosPort": "8084",
          "ClientPort": "8085"
        }
      },
      "PaxosType": "MultiPaxos",
      "FailureHandlingType": "None"
    }

This configuration file lists three different replicas, along with the
information about where they are running and on which ports. Also shown
is some configuration settings for which Paxos variant is to be run, as
well as the failure handling type, and logging information.

Note that the replicas are indexed by their id. We've already booted up
a replica with id 0, so we still have to boot up the remaining two
replicas on `pitter2` and `pitter3`.

    pitter2$ ./kvs -id 1
    pitter3$ ./kvs -id 2

Once all three replicas from the configuration file are booted up, the
replicas will begin communicating with one another, and client requests
can begin being sent to the service.

Logging
=======

Goxos and its sub packages use the glog package for logging. We recommend to
use the glog package when developing your own replicated service using Goxos.
Please see the [documentation][Godoc] for more information and available
command line flags. 

The glog package uses V-levels to control the amount of log output generated.
V-levels can only be specified for INFO, and not WARN, ERROR or FATAL. The glog
package also offers even more granularity by making it possible to specify
individual V-levels for specific files. Again, refer to the glog documentation
for details. 

The table below list the V-level logging policy used for Goxos and its example
applications.  

------------------------------------------------------------------------
  V-Level   Description
----------- ------------------------------------------------------------
     1      Logging of startup, initialization and shutdown procedures.

     2      Logging related to individual events, e.g. snapshotting,
	    failure handling etc.

     3      Logging for events related to individual Paxos slots
	    (instances). This can for example be logging when handling
	    Accept/Learn messages, executing application requests etc.

     4      Logging of low-level network and liveness procedures.
------------------------------------------------------------------------

[Godoc]: http://godoc.org/github.com/golang/glog
