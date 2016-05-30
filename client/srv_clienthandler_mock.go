package client

type ClientHandlerMock struct{}

func (chm *ClientHandlerMock) Start() {}

func (chm *ClientHandlerMock) Stop() {}

func (chm *ClientHandlerMock) ForwardResponse(resp *Response) {}
