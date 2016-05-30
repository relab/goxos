/*
Package multipaxos implements all of the functionality of the MultiPaxos protocol.
The implementation is split up into three main types: MultiProposer, MultiAcceptor, and
MultiLearner.

Each of the actors (MultiProposer, MultiAcceptor, MultiLearner) must implement the PaxosActor
interface:

    type PaxosActor interface {
      Start()
      Stop()
    }

The Start method gets called after the network is booted up, and is expected to register its
channels and spawn a goroutine waiting for results on the channels. The Stop method is expected
to stop said goroutine.

In addition to this interface, they have to implement their respective actor interface (one of
Proposer, Acceptor, or Learner).
*/
package multipaxos
