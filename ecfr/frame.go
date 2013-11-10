package ecfr

import (
	"fmt"
)

type Frame struct {
	Header    Header
	Datagrams []Datagram
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
		f.Datagrams = append(f.Datagrams, Datagram{})
		i := len(f.Datagrams) - 1

		b, err = f.Datagrams[i].Overlay(b)
		if err != nil {
			return
		}

		if f.Datagrams[i].Last() {
			break
		}
	}

	return
}
