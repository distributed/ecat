package ecfr

import (
	"errors"
	"fmt"
)

const (
	FrameOverheadLen = 2
)

type Frame struct {
	Header    Header
	Datagrams []*Datagram
	buffer    []byte
}

func (f *Frame) Overlay(d []byte) (b []byte, err error) {
	b, err = f.Header.Overlay(d)
	if err != nil {
		return
	}

	dgbl := f.Header.FrameLength()
	if int(dgbl) > len(b) {
		err = fmt.Errorf("frame expected %d bytes, only have %d", dgbl, len(b))
		return
	}

	for {
		f.Datagrams = append(f.Datagrams, &Datagram{})
		i := len(f.Datagrams) - 1

		b, err = f.Datagrams[i].Overlay(b)
		if err != nil {
			return
		}

		if f.Datagrams[i].Last() {
			break
		}
	}

	f.buffer = d

	return
}

func PointFrameTo(d []byte) (f Frame, err error) {
	if len(d) < FrameOverheadLen {
		err = errors.New("buffer too small to even contain frame header")
		return
	}

	d[0] = 0
	d[1] = 0
	_, err = f.Header.Overlay(d)
	if err != nil {
		return
	}

	f.buffer = d

	return
}

func (f *Frame) Commit() (d []byte, err error) {
	var incbuf []byte
	totlen := 0

	if len(f.Datagrams) == 0 {
		err = errors.New("ecat frame needs at least one datagram")
		return
	}

	clen := f.ByteLen()
	if clen > len(f.buffer) {
		err = fmt.Errorf("datagrams too long for frame, need %d, have %d", clen, len(f.buffer))
		return
	}

	lenmask := uint16((1 << 12) - 1)
	f.Header.Word &^= lenmask
	f.Header.Word |= uint16(clen-2) & lenmask

	incbuf, err = f.Header.Commit()
	if err != nil {
		return
	}
	totlen += len(incbuf)

	for _, dgram := range f.Datagrams {
		incbuf, err = dgram.Commit()
		if err != nil {
			return
		}
		totlen += len(incbuf)
	}

	d = f.buffer[0:totlen]

	return
}

func (f *Frame) ByteLen() int {
	clen := FrameOverheadLen
	for _, dgram := range f.Datagrams {
		clen += dgram.ByteLen()
	}
	return clen
}

func (f *Frame) NewDatagram(datalen int) (*Datagram, error) {
	curlen := f.ByteLen()
	maxlen := len(f.buffer)
	curfree := maxlen - curlen
	//fmt.Printf("curlen %d, maxlen %d, curfree %d\n", curlen, maxlen, curfree)
	if datalen <= curfree {
		dgram, err := PointDatagramTo(f.buffer[curlen:])
		if err != nil {
			return nil, err
		}

		err = dgram.SetDataLen(datalen)
		if err != nil {
			return nil, err
		}

		f.Datagrams = append(f.Datagrams, &dgram)

		return &dgram, nil
	}
	panic("datalen too high")
	return nil, errors.New("datalen too high")
}
