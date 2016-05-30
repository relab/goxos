# Goxos: A Go-based Paxos Implementation

Goxos was initially implemented to evaluate different failure handling
techniques for Paxos State Machines, including Replacement [1] and ARec [2].
The system has later been extended with other Paxos-variations.

### Old documentation

* [User guide](misc/doc/USERGUIDE.md)
* [Developer guide](misc/doc/DEVGUIDE.md)

### References

[1] L. Jehl, T. E. Lea, and H. Meling. _Replacement: The smart way to handle
failures in a replicated state machine._ In 34rd IEEE International Symposium
on Reliable Distributed Systems, SRDS ’15, pages 156–165. IEEE Computer
Society, 2015.

[2] L. Jehl and H. Meling. _Asynchronous Reconfiguration for Paxos State
Machines._ In M. Chatterjee, J.-n. Cao, K. Kothapalli, and S. Rajsbaum,
editors, Distributed Computing and Networking, volume 8314 of Lecture Notes in
Computer Science, pages 119–133. Springer Berlin Heidelberg, 2014.
