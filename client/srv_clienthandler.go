package client

type ClientHandler interface {
	Start()
	Stop()
	ForwardResponse(resp *Response)
}
