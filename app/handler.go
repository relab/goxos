package app

// The Handler interface must be implemented by an implementer of a Goxos replicated
// service. It must be possible to pass a command to the application through the Execute()
// method, as well as get and set the state of the application using GetState() and SetState().
type Handler interface {
	Execute(req []byte) (resp []byte)
	GetState(slotMarker uint) (sm uint, state []byte)
	SetState(state []byte) error
}
