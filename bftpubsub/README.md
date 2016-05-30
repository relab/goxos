bftpubsub
=========

This is an example application for testing the "Authenticated Broadcast" and "Reliable Broadcast" protocols using Goxos.

For more information, see \[1] and \[2].

Notes:

* There are two helper scripts for running each of the two protocols: ``runauthbc`` and ``runrelbc``.
* The publisher is set to publish a fixed amount of publications. The number of publications can be adjusted in the configuration file.
* All client, liveness and failure handling modules are disabled when using these two protocols.

\[1] Emil Haaland and Stian Ramstad. Tolerating Arbitrary Failures in a Pub/Sub System. Bachelor's thesis, 2014, University of Stavanger.

\[2] Leander Jehl and Hein Meling. Towards Byzantine Fault Tolerant Publish/Subscribe: A State Machine Approach. In Proceedings of the 9th Workshop on Hot Topics in Dependable Systems, HotDep '13, pages 5:1-5:5, New York, NY, USA, 2013. ACM.
