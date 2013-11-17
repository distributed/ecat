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

func ExecuteRead(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, n int, expwc uint16) (d []byte, err error) {
	return ExecuteReadOptions(c, ct, addr, n, expwc, Options{})
}

func ExecuteReadOptions(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, n int, expwc uint16, opts Options) (d []byte, err error) {
	nFrameLoss := 0

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

func ExecuteWrite(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, w []byte, expwc uint16) (err error) {
	return ExecuteWriteOptions(c, ct, addr, w, expwc, Options{})
}

func ExecuteWriteOptions(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, w []byte, expwc uint16, opts Options) (err error) {
	nFrameLoss := 0

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
