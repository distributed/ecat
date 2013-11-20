package ecmd

import (
	"errors"
	"fmt"
	"github.com/distributed/ecat/ecfr"
	"time"
)

type Commander interface {
	New(datalen int) (*ExecutingCommand, error)
	Cycle() error
	Close() error
}

type ExecutingCommand struct {
	DatagramOut *ecfr.Datagram

	DatagramIn *ecfr.Datagram
	Arrived    bool
	Overlayed  bool
	Error      error
}

var NoFrame = errors.New("frame did not arrive")
var NoOverlay = errors.New("failed to overlay")

type WorkingCounterError struct {
	Command    ecfr.CommandType
	Addr32     uint32
	Want, Have uint16
}

func (e WorkingCounterError) Error() string {
	return fmt.Sprintf("working counter error, want %d, have %d on %v %#08x", e.Want,
		e.Have,
		e.Command,
		e.Addr32)
}

func ChooseDefaultError(cmd *ExecutingCommand) error {
	if !cmd.Arrived {
		return NoFrame
	}

	if !cmd.Overlayed {
		return NoOverlay
	}

	return cmd.Error
}

func IsNoFrame(err error) bool {
	return err == NoFrame
}

func IsWorkingCounterError(err error) bool {
	_, ok := err.(WorkingCounterError)
	return ok
}

func ChooseWorkingCounterError(ec *ExecutingCommand, expwc uint16) error {
	havewc := ec.DatagramIn.WorkingCounter
	if expwc != havewc {
		return WorkingCounterError{
			ec.DatagramOut.Command,
			ec.DatagramOut.Addr32,
			expwc, havewc,
		}
	}

	return nil
}

const (
	DefaultFramelossTries = 3
)

type Options struct {
	FramelossTries int
	WCDeadline     time.Time
}

func (o Options) getFramelossTries() int {
	if o.FramelossTries == 0 {
		return DefaultFramelossTries
	}
	return o.FramelossTries
}
func (o Options) getWCDeadline() time.Time { return o.WCDeadline }

func ExecuteRead8(c Commander, addr ecfr.DatagramAddress, expwc uint16) (d uint8, err error) {
	return ExecuteRead8Options(c, addr, expwc, Options{})
}

func ExecuteRead8Options(c Commander, addr ecfr.DatagramAddress, expwc uint16, opts Options) (d uint8, err error) {
	var ds []byte
	ds, err = ExecuteRead(c, addr, 1, expwc)
	if err != nil {
		return
	}
	d = xgetUint8(ds)
	return
}

func ExecuteRead16(c Commander, addr ecfr.DatagramAddress, expwc uint16) (d uint16, err error) {
	return ExecuteRead16Options(c, addr, expwc, Options{})
}

func ExecuteRead16Options(c Commander, addr ecfr.DatagramAddress, expwc uint16, opt Options) (d uint16, err error) {
	var ds []byte
	ds, err = ExecuteRead(c, addr, 2, expwc)
	if err != nil {
		return
	}
	d = xgetUint16(ds)
	return
}

func ExecuteRead32(c Commander, addr ecfr.DatagramAddress, expwc uint16) (d uint32, err error) {
	return ExecuteRead32Options(c, addr, expwc, Options{})
}

func ExecuteRead32Options(c Commander, addr ecfr.DatagramAddress, expwc uint16, opt Options) (d uint32, err error) {
	var ds []byte
	ds, err = ExecuteRead(c, addr, 4, expwc)
	if err != nil {
		return
	}
	d = xgetUint32(ds)
	return
}

func ExecuteRead(c Commander, addr ecfr.DatagramAddress, n int, expwc uint16) (d []byte, err error) {
	return ExecuteReadOptions(c, addr, n, expwc, Options{})
}

func ExecuteReadOptions(c Commander, addr ecfr.DatagramAddress, n int, expwc uint16, opts Options) (d []byte, err error) {
	nFrameLoss := 0

	var ct ecfr.CommandType
	switch addr.Type() {
	case ecfr.Positional:
		ct = ecfr.APRD
	case ecfr.Fixed:
		ct = ecfr.FPRD
	case ecfr.Broadcast:
		ct = ecfr.BRD
	default:
		err = fmt.Errorf("ExecuteReadOptions: unsupported address type %v", addr.Type())
	}

	for {
		var ec *ExecutingCommand
		ec, err = c.New(n)
		if err != nil {
			return
		}

		dgo := ec.DatagramOut
		err = dgo.SetDataLen(n)
		if err != nil {
			return
		}

		dgo.Command = ct
		dgo.Addr32 = addr.Addr32()

		err = c.Cycle()
		if err != nil {
			return
		}

		err = ChooseDefaultError(ec)
		if err != nil {
			if IsNoFrame(err) {
				nFrameLoss++
				if nFrameLoss < opts.getFramelossTries() {
					continue
				}
			}
			return
		}

		err = ChooseWorkingCounterError(ec, expwc)
		if err != nil {
			now := time.Now()
			if now.Before(opts.getWCDeadline()) {
				continue
			}
		}

		d = ec.DatagramIn.Data()
		return
	}

	panic("not reached")
}

func ExecuteWrite8(c Commander, addr ecfr.DatagramAddress, w uint8, expwc uint16) (err error) {
	return ExecuteWrite8Options(c, addr, w, expwc, Options{})
}

func ExecuteWrite8Options(c Commander, addr ecfr.DatagramAddress, w uint8, expwc uint16, opts Options) (err error) {
	ws := make([]byte, 1)
	putUint8(ws, w)
	return ExecuteWriteOptions(c, addr, ws, expwc, opts)
}

func ExecuteWrite16(c Commander, addr ecfr.DatagramAddress, w uint16, expwc uint16) (err error) {
	return ExecuteWrite16Options(c, addr, w, expwc, Options{})
}

func ExecuteWrite16Options(c Commander, addr ecfr.DatagramAddress, w uint16, expwc uint16, opts Options) (err error) {
	ws := make([]byte, 2)
	putUint16(ws, w)
	return ExecuteWriteOptions(c, addr, ws, expwc, opts)
}

func ExecuteWrite32(c Commander, addr ecfr.DatagramAddress, w uint32, expwc uint16) (err error) {
	return ExecuteWrite32Options(c, addr, w, expwc, Options{})
}

func ExecuteWrite32Options(c Commander, addr ecfr.DatagramAddress, w uint32, expwc uint16, opts Options) (err error) {
	ws := make([]byte, 4)
	putUint32(ws, w)
	return ExecuteWriteOptions(c, addr, ws, expwc, opts)
}

func ExecuteWrite(c Commander, addr ecfr.DatagramAddress, w []byte, expwc uint16) (err error) {
	return ExecuteWriteOptions(c, addr, w, expwc, Options{})
}

func ExecuteWriteOptions(c Commander, addr ecfr.DatagramAddress, w []byte, expwc uint16, opts Options) (err error) {
	nFrameLoss := 0

	var ct ecfr.CommandType
	switch addr.Type() {
	case ecfr.Positional:
		ct = ecfr.APWR
	case ecfr.Fixed:
		ct = ecfr.FPWR
	case ecfr.Broadcast:
		ct = ecfr.BWR
	default:
		err = fmt.Errorf("ExecuteWriteOptions: unsupported address type %v", addr.Type())
	}

	for {
		var ec *ExecutingCommand
		ec, err = c.New(len(w))
		if err != nil {
			return
		}

		dgo := ec.DatagramOut
		err = dgo.SetDataLen(len(w))
		if err != nil {
			return
		}
		copy(dgo.Data(), w)

		dgo.Command = ct
		dgo.Addr32 = addr.Addr32()

		err = c.Cycle()
		if err != nil {
			return
		}

		err = ChooseDefaultError(ec)
		if err != nil {
			if IsNoFrame(err) {
				nFrameLoss++
				if nFrameLoss < opts.getFramelossTries() {
					continue
				}
			}
			return
		}

		err = ChooseWorkingCounterError(ec, expwc)
		if err != nil {
			now := time.Now()
			if now.Before(opts.getWCDeadline()) {
				continue
			}
		}

		return
	}

}
