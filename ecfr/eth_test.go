package ecfr

import (
	"reflect"
	"testing"
)

func makeEmptyFrameBuffer() []byte {
	return make([]byte, min_framelen_with_fcs)
}

func TestETHFramePayloadOps(t *testing.T) {
	{
		tlb := make([]byte, 20)
		_, err := OverlayETHFrame(tlb)
		if err == nil {
			t.Fatalf("OverlayETHFrame did not fail on buffer too small to contain ETH frame")
		}
	}

	buf := make([]byte, min_framelen_with_fcs)
	ef, err := OverlayETHFrame(buf)
	if err != nil {
		t.Fatalf("OverlayETHFrame should have worked, returned err %v", err)
	}

	{
		pl := ef.GetPayload()
		exppllen := len(buf) - 14 - 4 // avoid normal constant encoding
		if len(pl) != exppllen {
			t.Fatalf("for packet with frame buf size %d, expected %d payload bytes, got  %d", len(buf), exppllen, len(pl))
		}
	}

	{
		// maximum length payload
		// 4 bytes pl[len(pl):cap(pl)] should include FCS
		pl := ef.GetPayload()

		if cap(pl)-len(pl) != fcs_len {
			t.Fatalf("full size payload should have capacity for %d bytes of FCS, only have %d", fcs_len, cap(pl)-len(pl))
		}

		// we got a maximally big payload, it should not be possible to make
		// it any bigger.
		err := ef.SetPayloadLen(len(pl) + 1)
		if err == nil {
			t.Fatalf("increasing the size of a maximally sized pl did not yield an error!")
		}

		// try to set a size too small to go without padding
		err = ef.SetPayloadLen(46 - 1) // avoid constants, again
		if err == nil {
			t.Fatalf("setting the payload too small did not yield an error!")
		}
	}
}

func TestETHFrameDecoding(t *testing.T) {
	hdrbytes := []byte{0xab, 0xcd, 0xef, 0x12, 0x23, 0x34, 0xde, 0xad, 0xbe, 0xef, 0xaa, 0x55, 0x88, 0xa2}
	fb := makeEmptyFrameBuffer()
	copy(fb, hdrbytes)
	ef, err := OverlayETHFrame(fb)
	if err != nil {
		t.Fatalf("overlaying should work on this header,failed with %v\n", err)
	}

	wantdest := ETHAddr{0xab, 0xcd, 0xef, 0x12, 0x23, 0x34}
	if !reflect.DeepEqual(ef.Destination, wantdest) {
		t.Fatalf("destination address does not match, want %v, got %v", wantdest, ef.Destination)
	}

	wantsrc := ETHAddr{0xde, 0xad, 0xbe, 0xef, 0xaa, 0x55}
	if !reflect.DeepEqual(ef.Source, wantsrc) {
		t.Fatalf("destination address does not match, want %v, got %v", wantsrc, ef.Source)
	}

	wantethtype := uint16(0x88a2)
	if ef.Type != wantethtype {
		t.Fatalf("want eth type %#04x, got %#04x", wantethtype, ef.Type)
	}
}
