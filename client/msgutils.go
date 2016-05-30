package client

import (
	"bytes"
	"fmt"
	"hash/fnv"
)

// the length of our ID field (base64 encoded SHA-1 hash)
const validIDLength = 28

var h = fnv.New32a()

// Display the Request in truncated string form.
func (r *Request) SimpleString() string {
	return fmt.Sprintf("request from %v with seq %d", r.GetId(), r.GetSeq())
}

// Display the Request in its full glory.
func (r *Request) FullString() string {
	h.Reset()
	h.Write(r.GetVal())
	return fmt.Sprintf("%v and value hash %x", r.SimpleString(), h.Sum32())
}

// Check whether two Requests are equal.
func (r *Request) Equal(other Request) bool {
	return r.GetType() == other.GetType() &&
		r.GetId() == other.GetId() &&
		r.GetSeq() == other.GetSeq() &&
		bytes.Compare(r.GetVal(), other.GetVal()) == 0
}

// Do a FNV hash of a Request.
func (r *Request) Hash() uint32 {
	h := fnv.New32a()
	h.Write(r.GetVal())
	return h.Sum32()
}

// IsInvalid returns true if the request's ID field is invalid.
func (r *Request) IsIDInvalid() bool {
	return r.GetId() == "" || len(r.GetId()) != validIDLength
}

// Display the Response in truncated string form.
func (r *Response) SimpleString() string {
	if !r.HasError() {
		return fmt.Sprintf("response to %v with seq %d", r.GetId(), r.GetSeq())
	}

	return fmt.Sprintf("response to %v with seq %d and error code %v",
		r.GetId(), r.GetSeq(), r.GetErrorCode())
}

// Display the Response in its fully glory.
func (r *Response) FullString() string {
	if !r.HasError() {
		h.Reset()
		h.Write(r.GetVal())
		return fmt.Sprintf("%v, value hash %x", r.SimpleString(), h.Sum32())
	}

	return fmt.Sprintf("%v, error detail: %v", r.SimpleString(), r.GetErrorDetail())
}

// Check whether the response contains an error.
func (r *Response) HasError() bool {
	return r.GetErrorCode() != 0
}
