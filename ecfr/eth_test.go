package ecfr

import (
	"testing"
)

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
