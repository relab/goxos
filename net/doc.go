/*
Package net is responsible for abstracting away all of the inter-replica
communication in the system. This module doesn't handle communication between replica
and client.

This package provides a Demuxer and a Sender. The Demuxer is responsible for handling
incoming connections, from which it receives messages and passes them on to the interested
parties. The Sender is responsible for establishing connections to other replicas, as well
as the sending of messages to other replicas.

For the demuxer to know which parties are interested in receiving different types of messages,
they must be first registered before the networking subsystem starts up:

    prepareChan := make(chan px.PrepareMsg, 8)
    a.dmx.RegisterChannel(prepareChan)

The communication from the Demuxer to an actor happens through the use of channels, as shown
above. First a channel is created by the actor, in this case the channel is for receiving PrepareMsg
messages. Then, the RegisterChannel method is called with the channel. This tells the Demuxer
that if it receives any messages of the specified type, it will send it over this channel. Multiple
channels of the same type may be registered with the Demuxer.

NB: The channels should all be registered before the networking subsystem is started up, no more
channels should be registered after. This is due to the fact that the GxConnection goroutines pass
messages to the Demuxer, and if channels are registered after network start-up, bad things could
happen.
*/
package net
