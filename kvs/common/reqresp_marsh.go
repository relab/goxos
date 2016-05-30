package common

import (
	"bufio"
	"encoding/binary"
	"io"
	"sync"
)

type byteReader interface {
	io.Reader
	ReadByte() (c byte, err error)
}

func (t *MapRequest) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type MapRequestCache struct {
	mu    sync.Mutex
	cache []*MapRequest
}

func NewMapRequestCache() *MapRequestCache {
	c := &MapRequestCache{}
	c.cache = make([]*MapRequest, 0)
	return c
}

func (p *MapRequestCache) Get() *MapRequest {
	var t *MapRequest
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &MapRequest{}
	}
	return t
}
func (p *MapRequestCache) Put(t *MapRequest) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *MapRequest) Marshal(wire io.Writer) {
	var b [10]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.Ct)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Key))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		bs = b[:1]
		bs[0] = byte(t.Key[i])
		wire.Write(bs)
	}
	bs = b[:]
	alen2 := int64(len(t.Value))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		bs = b[:1]
		bs[0] = byte(t.Value[i])
		wire.Write(bs)
	}
}

func (t *MapRequest) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [10]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.Ct = CommandType(bs[0])
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Key = make([]byte, alen1)
	for i := int64(0); i < alen1; i++ {
		if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
			return err
		}
		t.Key[i] = byte(bs[0])
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Value = make([]byte, alen2)
	for i := int64(0); i < alen2; i++ {
		if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
			return err
		}
		t.Value[i] = byte(bs[0])
	}
	return nil
}

func (t *MapResponse) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type MapResponseCache struct {
	mu    sync.Mutex
	cache []*MapResponse
}

func NewMapResponseCache() *MapResponseCache {
	c := &MapResponseCache{}
	c.cache = make([]*MapResponse, 0)
	return c
}

func (p *MapResponseCache) Get() *MapResponse {
	var t *MapResponse
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &MapResponse{}
	}
	return t
}
func (p *MapResponseCache) Put(t *MapResponse) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *MapResponse) Marshal(wire io.Writer) {
	var b [10]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.ToType)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Value))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		bs = b[:1]
		bs[0] = byte(t.Value[i])
		wire.Write(bs)
	}
	bs[0] = byte(t.Found)
	wire.Write(bs)
	bs = b[:]
	alen2 := int64(len(t.Err))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		bs = b[:1]
		bs[0] = byte(t.Err[i])
		wire.Write(bs)
	}
}

func (t *MapResponse) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [10]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.ToType = CommandType(bs[0])
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Value = make([]byte, alen1)
	for i := int64(0); i < alen1; i++ {
		if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
			return err
		}
		t.Value[i] = byte(bs[0])
	}
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.Found = byte(bs[0])
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Err = make([]byte, alen2)
	for i := int64(0); i < alen2; i++ {
		if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
			return err
		}
		t.Err[i] = byte(bs[0])
	}
	return nil
}
