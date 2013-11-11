package ecmd

import (
	"errors"
	"fmt"
	"github.com/distributed/ecat/ecfr"
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

func ExecuteRead(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, n int, expwc uint16) (d []byte, err error) {
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
		return
	}

	err = ChooseWorkingCounterError(ec, expwc)
	if err != nil {
		return
	}

	d = ec.DatagramIn.Data()
	return
}

func ExecuteWrite(c Commander, ct ecfr.CommandType, addr ecfr.DatagramAddress, w []byte, expwc uint16) (err error) {
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
		return
	}

	err = ChooseWorkingCounterError(ec, expwc)
	if err != nil {
		return
	}

	return
}
