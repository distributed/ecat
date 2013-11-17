package sim

import (
	"github.com/distributed/ecat/ecfr"
)

const (
	maxDatagramsLen = 1470
)

type L2Bus struct {
	oframes []*ecfr.Frame

	Slaves []FrameProcessor
}

func (b *L2Bus) New(maxdatalen int) (fr *ecfr.Frame, err error) {
	var vframe ecfr.Frame
	buf := make([]byte, maxDatagramsLen+ecfr.FrameOverheadLen)
	vframe, err = ecfr.PointFrameTo(buf)
	if err != nil {
		return
	}

	vframe.Header.SetType(1)

	fr = &vframe
	b.oframes = append(b.oframes, fr)
	return
}

func (b *L2Bus) Cycle() (iframes []*ecfr.Frame, err error) {
	defer func() {
		b.oframes = nil
	}()

	for i, oframe := range b.oframes {
		var obytes []byte

		obytes, err = oframe.Commit()
		if err != nil {
			return
		}

		_ = i
		//fmt.Printf("oframe #%d: %s", i, oframe.MultilineSummary())

		coframe := new(ecfr.Frame)
		cbytes := make([]byte, len(obytes))
		copy(cbytes, obytes)
		_, err = coframe.Overlay(cbytes)
		if err != nil {
			return
		}

		for _, slave := range b.Slaves {
			coframe = slave.ProcessFrame(coframe)
			if coframe == nil {
				break
			}
		}

		if coframe != nil {
			iframes = append(iframes, coframe)
		}
	}

	for i, iframe := range iframes {
		_, _ = i, iframe
		//fmt.Printf("iframe #%d: %s", i, iframe.MultilineSummary())
	}

	return
}

func (b *L2Bus) Close() error { return nil }
