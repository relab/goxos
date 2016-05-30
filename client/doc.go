/*
Package client is responsible for client-replica communication. In Goxos, a client
can connect to the service and send requests. These requests will usually be run by the
service, generating a response, which is sent back to the client.

Clients are handled through the use of ClientConn and ClientHandler. The ClientHandler
listens on a socket for connections from clients, does a handshake with the client, and
then creates a ClientConn -- which is responsible for further handling of each specific
client.

To recompile the Protobuf msg.proto file, use the command: protoc msg.proto --go_out=.
*/
package client
