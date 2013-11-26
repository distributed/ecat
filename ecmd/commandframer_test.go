package ecmd

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/distributed/ecat/ecfr"
	"reflect"
	"testing"
)

type expectedCFScheduling struct {
	dgramslist [][]*ecfr.Datagram
}

func TestCommandFramerScheduling(t *testing.T) {
	type cfSchedulingPairs struct {
		lens                 []int
		expectedCFScheduling expectedCFScheduling
	}

	pairs := []cfSchedulingPairs{
		cfSchedulingPairs{[]int{6}, expectedCFScheduling{
			[][]*ecfr.Datagram{
				[]*ecfr.Datagram{makeLenDgram(6, 0, true)}}}},
		cfSchedulingPairs{[]int{22, CommandFramerMaxDatagramsLen - ecfr.DatagramOverheadLength}, expectedCFScheduling{
			[][]*ecfr.Datagram{
				[]*ecfr.Datagram{makeLenDgram(22, 0, true)},
				[]*ecfr.Datagram{makeLenDgram(CommandFramerMaxDatagramsLen-ecfr.DatagramOverheadLength, 1, true)}}}},
		cfSchedulingPairs{[]int{128, 96}, expectedCFScheduling{
			[][]*ecfr.Datagram{
				[]*ecfr.Datagram{makeLenDgram(128, 0, false), makeLenDgram(96, 0, true)}}}},
		cfSchedulingPairs{[]int{140, 65, 1400}, expectedCFScheduling{
			[][]*ecfr.Datagram{
				[]*ecfr.Datagram{makeLenDgram(140, 0, false), makeLenDgram(65, 0, true)},
				[]*ecfr.Datagram{makeLenDgram(1400, 1, true)}}}},
	}

	for i, pair := range pairs {
		f := &oneshotFramer{}
		cf := NewCommandFramer(f)

		for _, l := range pair.lens {
			_, err := cf.New(l)
			if err != nil {
				t.Fatalf("case %d: did not expect New() to fail. err is %v", i, err)
			}
		}

		err := cf.Cycle()
		if err != nil {
			t.Fatalf("did not expect Cycle() to fail. err is %v", err)
		}

		dgramslist := pair.expectedCFScheduling.dgramslist
		if len(f.frames) != len(dgramslist) {
			t.Fatalf("case %d: expected %d frames, got %d", i, len(dgramslist), len(f.frames))
		}

		for j, frame := range f.frames {
			dgrams := dgramslist[j]
			if len(frame.Datagrams) != len(dgrams) {
				t.Fatalf("case %d, frame %d: expected %d datagrams, got %d", i, j, len(dgrams), len(frame.Datagrams))
			}

			for k, dgram := range frame.Datagrams {
				expdgram := dgrams[k]

				if !reflect.DeepEqual(expdgram, dgram) {
					spew.Dump(expdgram)
					spew.Dump(dgram)
					t.Fatalf("case %d, frame %d, dgram %d: expected %v, got %v\n", i, j, k, expdgram, dgram)
				}
			}
		}

	}
}

type oneshotFramer struct {
	frames []*ecfr.Frame
	cycled bool
}

func (f *oneshotFramer) New(maxdatalen int) (*ecfr.Frame, error) {
	b := make([]byte, maxdatalen+ecfr.FrameOverheadLen)
	var frame ecfr.Frame
	var err error
	frame, err = ecfr.PointFrameTo(b)
	if err != nil {
		return nil, err
	}

	f.frames = append(f.frames, &frame)
	return &frame, nil
}

func (f *oneshotFramer) Cycle() ([]*ecfr.Frame, error) {
	if !f.cycled {
		return f.frames, nil
	}
	panic("oneshotFramer was already cycled")
}

func makeLenDgram(plen int, index uint8, last bool) *ecfr.Datagram {
	ub := make([]byte, plen+ecfr.DatagramOverheadLength)
	dgram, err := ecfr.PointDatagramTo(ub)
	if err != nil {
		panic(err)
	}
	err = dgram.SetDataLen(plen)
	if err != nil {
		panic(err)
	}

	dgram.Index = index
	dgram.SetLast(last)

	return &dgram
}
