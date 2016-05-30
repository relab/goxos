% Goxos Developer's Guide
% Stephen Michael Jothen
% June 27, 2013

The Goxos User's Guide goes through the steps of creating a replicated
service using Goxos. This document, the Goxos Developer's Guide, does
something a little different. It explains how to modify Goxos to support
future Paxos variants.

In order to do this, we will go through the steps of developing the
framework for an implementation of Fast Paxos. Most of the actual
implementation details will be left out of this guide, and we will focus
on the main changes required to Goxos to support such a new variant of
Paxos.

Actor Interfaces
================

The first key to understanding how to develop a new Paxos variant in
Goxos is to look at the requirements. The main requirements are that
there are three actors -- a proposer, an acceptor, and a learner. This
does not mean that all three must be functioning, in fact there can be a
single functioning actor as is done in the implementation of Acropolis.

Each actor has a specified set of methods that the actor must implement.
In order to enforce that they implement the correct set of methods, we
define a set of interfaces for each actor.

    type PaxosActor interface {
      Start()
      Stop()
    }

    type Proposer interface {
      PaxosActor
      HandlePromise(PromiseMsg) error
      HandleVal(Value) error
    }

Each actor must implement the `PaxosActor` interface. This is done by
embedding the `PaxosActor` interface in each of the actor interfaces.
This is shown above. This forces each of the actors to define `Start`
and `Stop` methods which tell the actor to start and stop handling
messages from channels respectively. In addition to these methods, each
actor must implement a specific set of methods for itself. In the case
of the proposer above, these are the `HandlePromise` and `HandleVal`
methods. These methods are defined in the `Proposer` interface because
the proposer in Paxos typically receives two types of messages, promise
messages from the acceptors, as well as client requests.

We can also take a look at the interfaces for the other two actors in
Paxos, the acceptor and the learner:

    type Acceptor interface {
      PaxosActor
      HandlePrepare(PrepareMsg) error
      HandleAccept(AcceptMsg) error
      GetState(afterSlot SlotId, release <-chan bool) *AcceptorSlotMap
      SetState(slotmap *AcceptorSlotMap)
    }

    type Learner interface {
      PaxosActor
      HandleLearn(LearnMsg) error
    }

The `Learner` interface is rather simple, stating that the Learner must
implement a `HandleLearn` method in addition to the base `PaxosActor`
methods. This mirrors the definition of Paxos, which says that a learner
is responsible for handling incoming learn messages from the acceptors.
The `Acceptor` interface is a bit more complicated. The acceptor in
traditional Paxos handles two types of messages, a prepare message and
an accept message, both sent from the leader. This is the reason for the
requirement of the `Acceptor` needing to define `HandlePrepare` and
`HandlePromise` methods with the given signatures.

In addition to these methods, the acceptor actor must also implement
`GetState` and `SetState` methods, which are currently used by the two
supported failure handling methods, Live Replacement and
Reconfiguration.

Actor Implementation
====================

Now that we understand the actor interfaces that each actor is required
to implement, we can begin to discuss the actual implementation of the
actors. In our case, we are implementing Fast Paxos. Because of this, we
will refer to our actors as `FastProposer`, `FastAcceptor` and
`FastLearner`.

The first thing to do is to create the necessary types and methods that
will fulfill the interface requirements. This is quite simple. First we
will implement the boilerplate code for the `FastProposer`:

    import (
      px "goxos/paxos"
    )

    type FastProposer struct {
    }

    func (fp *FastProposer) Start() {
    }

    func (fp *FastProposer) Stop() {
    }

    func (fp *FastProposer) HandlePromise(msg px.PromiseMsg) error {
    }

    func (fp *FastProposer) HandleVal(val px.Value) error {
    }

This satisfies the requirements, however, it is also useful to define a
constructor that initializes the actor based on a `PaxosPack` struct,
which we'll go into further detail later on:

    func NewFastProposer(pp px.PaxosPack) *FastProposer {
      // ...
    }

Note that the `FastProposer` has implemented all of the methods required
by the `Proposer` interface, as well as a constructor. The same pattern
can be done for the other two actors, the `FastAcceptor` and
`FastLearner`.

The `PaxosPack` that each actor constructor gets is shown below:

    type PaxosPack struct {
      Id          grp.Id
      Gm          *grp.GrpMgr
      StopCheckIn *sync.WaitGroup
      EventLogger *elog.EventLogger

      Alpha         uint
      NrOfNodes     uint
      NrOfAcceptors uint

      BatchPaxosTimeout uint
      BatchPaxosMaxSize uint

      RunProp bool
      RunAcc  bool
      RunLrn  bool

      Dmx *network.Demuxer

      TrustP <-chan grp.Id
      TrustA <-chan grp.Id
      TrustL <-chan grp.Id

      Ucast  chan<- network.Packet
      Bcast  chan<- interface{}
      BcastP chan<- interface{}
      BcastA chan<- interface{}
      BcastL chan<- interface{}

      PropChan        <-chan Value // Proposal values
      DcdChan         chan<- Value // Decided values
      DcdSlotIdToProp chan SlotId

      FirstSlot       SlotId
      NextExpectedDcd SlotId
    }

As can be seen, the struct is quite large. This struct gets passed to
each of the actor constructors, and the constructors can pull out the
necessary information that it needs. Take, for example, the `TrustP`,
`TrustA`, and `TrustL` channels. These channels are created by the
server module and allow the proposer, acceptor, and learner actors to
retrieve information about leader election. The `Ucast` and `Bcast`
channels are used by an actor to send information to a single replica or
all of the replicas respectively. Other interesting things in the
`PaxosPack` are the `PropChan` and `DcdChan`, which are where the client
requests are retrieved from and where decided values are placed. Also of
note is the `Dmx`, containing the demuxer which actors can use to
register their interest in certain message types. Now, we can take a
look at the actual code in the `NewFastProposer` constructor:

    func NewFastProposer(pp px.PaxosPack) *FastProposer {
      return &FastProposer{
        id:          pp.Id,
        dmx:         pp.Dmx,
        nAcceptors:  pp.NrOfAcceptors,
        nFaulty:     (pp.NrOfAcceptors - 1) / 3,
        nextSlot:    pp.NextExpectedDcd,
        ucast:       pp.Ucast,
        bcast:       pp.Bcast,
        trust:       pp.TrustP,
        slots:       px.NewProposerSlotMap(),
        stop:        make(chan bool),
        stopCheckIn: pp.StopCheckIn,
      }
    }

As expected, the `FastProposer` simply copies a lot of the values from
the `PaxosPack` into its own structure, which it can access from any of
the methods defined on it, as we discussed earlier.

The implementation of the `HandlePromise` and `HandleVal` methods will
not be discussed in this document, since this is giving a more general
picture of how a Paxos variant is implemented; we are not interested in
the actual details of the Fast Paxos algorithm itself.

However, we can take a look at the `Start` and `Stop` methods to see how
they are implemented for Fast Paxos, since the same techniques are used
for all of the Paxos variants implemented in Goxos.

First, we will look at the `Start` method shown below.

    func (fp *FastProposer) Start() {
      fp.registerChannels()

      go func() {
        defer fp.stopCheckIn.Done()
        for {
          select {
          case decided := <-fp.decidedChan:
            fp.handleDecided(decided)
          case promise := <-fp.promiseChan:
            fp.HandlePromise(promise)
          case cpromise := <-fp.compPromiseChan:
            fp.HandleCompositePromise(cpromise)
          case trustId := <-fp.trust:
            fp.changeLeader(trustId)
          case <-fp.stop:
            return
          }
        }
      }()
    }

The first thing this method does is call the method `registerChannel`:

    func (fp *FastProposer) registerChannels() {
      decidedChan := make(chan px.SlotId)
      fp.decidedChan = decidedChan
      fp.dmx.RegisterChannel(decidedChan)

      promiseChan := make(chan px.PromiseMsg)
      fp.promiseChan = promiseChan
      fp.dmx.RegisterChannel(promiseChan)

      compPromiseChan := make(chan px.CompositePromiseMsg)
      fp.compPromiseChan = compPromiseChan
      fp.dmx.RegisterChannel(compPromiseChan)
    }

This method creates three channels, stores them in the `FastProposer`
structure, and then registers each channel with the demuxer. By calling
`RegisterChannel` on the demuxer, the demuxer will infer the type of
channel and place any messages of that type that it receives over the
network on said channel. For example, a `PromiseMsg` channel is
registered. If any `PromiseMsg` types are received over the network,
they will be placed on this channel.

Once the registering of channels is complete, the `Start` method spawns
a goroutine using the `go` keyword. This anonymous goroutine is the most
central part of Goxos's concurrency scheme. As can be seen in the code,
the goroutine is responsible for handling all of the messages received
over the channels -- both the channels registered in the
`registerChannels` method, and the channels that were created in the
constructor. A common idiom used in Goxos is to receive a message from
one of the registered channels, and then pass it on to a method with a
`Handle` prefix. These `Handle` methods are responsible for doing the
heavy lifting of the actual implemented protocol. One channel to take
note of is the `stop` channel, which is created by the constructor, and
is used to cause the goroutine to exit.

    func (fp *FastProposer) Stop() {
      fp.stop <- true
    }

The `stop` channel is read from in the anonymous goroutine created by
`Start`, and is written to by the `Stop` method, which places a value on
the channel. This `Stop` method is used by the server module to cause
all message handling for this actor to cease.

The final thing to cover about the implementation of a Paxos variant is
how to send messages to other replicas. Usually a message is read off
one of the channels and handled by one of the `Handle` methods. This
method may want to act on the message, perhaps sending a response back
to the sender or broadcasting to the whole replicated service. This can
be done through use of the `ucast` and `bcast` channels which were taken
from the `PaxosPack` in the constructor.

The simplest way of doing this is to abstract the channel communication
away for each of the actors. This can be done by defining `send` and
`broadcast` methods on the actor as follows:

    func (fa *FastProposer) send(msg interface{}, id grp.Id) {
        fa.ucast <- network.Packet{DestId: id, Data: msg}
    }

    func (fa *FastProposer) broadcast(msg interface{}) {
        fa.bcast <- msg 
    }

Now, in any of the `Handle` methods, the actor can simply call the
desired method to communicate with the outside world. If it wants to
send to a specific replica, it can use the `send` method, supplying the
message as well as the id of the receiving replica. To broadcast, a
message is simply supplied to the `broadcast` method.

With the blueprint of implementing an actual actor covered, we can begin
discussing how the construction and initialization of the actors occurs.

Configuration
=============

When implementing a Paxos variant, it is often useful to be able to
configure which Paxos variant is to be run from the configuration file.
To do this, some of the configuration parsing code must be edited.

The necessary code to change is located in the `config/variants.go`
file:

    const (
      MultiPaxos PaxosVariant = iota
      FastPaxos
      BatchPaxos
      MultiPaxosLr
    )

    var paxosVariantMap = map[string]PaxosVariant{
      "FastPaxos":    FastPaxos,
      "MultiPaxos":   MultiPaxos,
      "BatchPaxos":   BatchPaxos,
      "MultiPaxosLr": MultiPaxosLr,
    }

    var paxosVariantToStringMap = map[PaxosVariant]string{
      FastPaxos:    "FastPaxos",
      MultiPaxos:   "MultiPaxos",
      BatchPaxos:   "BatchPaxos",
      MultiPaxosLr: "MultiPaxosLr",
    }

In order to add a variant that you can set in the configuration file, a
new constant representing the variant must be added, as well as a
mapping from string to the constant, and a mapping from the constant to
a string. This variant constant can then be used when instantiation of
the variant occurs in the server module, which we will discuss next.

Actor Construction and Initialization
=====================================

We've already discussed how to implement the types and the methods
required by each of the actors. However, these types still need to be
used by the Goxos framework. Who actually calls the `NewFastProposer`
constructor and calls the `Start` method? The server module is
responsible for much of the setup of Goxos, including this.

The actual initialization of much of the Goxos framework occurs in the
server module, specifically in the `server/init.go` file. The
`initPaxos` method in particular is the most interesting for
implementors of Paxos variants. This function is responsible for
creating the actual `PaxosPack`, as well as the initialization of the
variants:

    func (s *Server) initPaxos() {
      node, exists := s.config.Nodes.LookupNode(s.id)

      if !exists {
        panic("server: can't find self in nodemap")
      }

      propLdChan := s.ld.SubscribeToLdMsgs("proposer")
      accLdChan := s.ld.SubscribeToLdMsgs("acceptor")
      lrnLdChan := s.ld.SubscribeToLdMsgs("learner")

      pp := paxos.PaxosPack{
        // ...
      }

      switch s.config.PaxosType {
      case config.MultiPaxos:
        log.Println("server: running in multipaxos mode")
        s.prop, s.acc, s.lrn = multipaxos.CreateMultiPaxos(pp)
      case config.FastPaxos:
        log.Println("server: running in fastpaxos mode")
        s.prop, s.acc, s.lrn = fastpaxos.CreateFastPaxos(pp)
      case config.BatchPaxos:
        log.Println("server: running in batchpaxos mode")
        s.prop, s.acc, s.lrn = batchpaxos.CreateBatchPaxos(pp)
      case config.MultiPaxosLr:
        log.Println("server: running in multipaxos-lr mode")
        s.prop, s.acc, s.lrn = multipaxoslr.CreateMultiPaxosLr(pp)
      default:
        panic("unknown paxos type")
      }
    }

We can also see that, depending on the Paxos variant type set in the
configuration file, we instantiate a different Paxos variant with the
`PaxosPack`. Since we have to instatiate three actors for each of the
variants, each variant is required to have a super constructor that
constructs and returns all of the actors. Take, for example, the Fast
Paxos constructor shown below:

    func CreateFastPaxos(pp px.PaxosPack)
        (*FastProposer, *FastAcceptor, *FastLearner) {
      p := NewFastProposer(pp)
      a := NewFastAcceptor(pp)
      l := NewFastLearner(pp)

      return p, a, l
    }

All that is done in the super constructor is to call each of the actor
constructors in turn, and then return them in the specified order.
Usually the super constructor should be placed in a separate file, for
example `create.go` in the Paxos variant's module.

With all of this out of the way, you're now ready to create your own
Goxos implementation of a Paxos variant. Good luck!
