/*
Package paxos provides some Paxos-related structures that are shared by some of the
different Paxos modules.

This includes things like messages which are shared between Fast Paxos, MultiPaxos,
MultiPaxosLr. It also includes slot maps used by the various actors of the Paxos
variants, and basic utility functions.
*/
package paxos
