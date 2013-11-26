package ecmd

import (
	"errors"
	"github.com/distributed/ecat/ecfr"
)

const (
	CommandFramerMaxDatagramsLen = 1470
)

type outgoingFrame struct {
	frame *ecfr.Frame
	cmds  []*ExecutingCommand
}

type CommandFramer struct {
	currentIndex uint8

	frameOpen          bool
	currentFrame       *ecfr.Frame
	currentFrameLen    uint16
	currentFrameOffset uint16
	currentDgram       *ecfr.Datagram
	currentCmds        []*ExecutingCommand

	frameQueue []outgoingFrame

	inFrameQueue []*ecfr.Frame

	framer Framer
}

func NewCommandFramer(framer Framer) *CommandFramer {
	return &CommandFramer{framer: framer}
}

func (cf *CommandFramer) New(datalen int) (*ExecutingCommand, error) {
	var err error

	dbgl := datalen + ecfr.DatagramOverheadLength
	if dbgl > CommandFramerMaxDatagramsLen {
		return nil, errors.New("datalen exceeds maximum datagram length")
	}

	if cf.frameOpen {
		if dbgl > int(cf.currentFrameLen-cf.currentFrameOffset) {
			cf.finishFrame()
			err = cf.newFrame()
			if err != nil {
				return nil, err
			}

		}
	} else {
		err = cf.newFrame()
		if err != nil {
			return nil, err
		}
	}

	var dg *ecfr.Datagram
	//fmt.Printf("want NewDatagram datalen %d\n", datalen)
	dg, err = cf.currentFrame.NewDatagram(datalen)
	if err != nil {
		return nil, err
	}
	cf.currentDgram = dg

	cf.currentFrameOffset += uint16(dbgl)

	cmd := &ExecutingCommand{
		DatagramOut: dg,
	}
	cf.currentCmds = append(cf.currentCmds, cmd)
	return cmd, nil
}

func (cf *CommandFramer) finishFrame() {
	if len(cf.currentFrame.Datagrams) > 0 {
		for i := 0; i < len(cf.currentFrame.Datagrams)-1; i++ {
			cf.currentFrame.Datagrams[i].SetLast(false)
		}
		cf.currentFrame.Datagrams[0].Index = cf.currentIndex
		cf.currentFrame.Datagrams[len(cf.currentFrame.Datagrams)-1].SetLast(true)
		cf.frameQueue = append(cf.frameQueue, outgoingFrame{cf.currentFrame, cf.currentCmds})
	}

	cf.frameOpen = false
	cf.currentFrame = nil
	cf.currentFrameLen = 0
	cf.currentFrameOffset = 0xffff
	cf.currentCmds = nil
	cf.currentIndex++
}

func (cf *CommandFramer) newFrame() error {
	// TODO: constant for ecat frame header len (2)

	var (
		frame *ecfr.Frame
		err   error
	)

	/*buf := make([]byte, CommandFramerMaxDatagramsLen+2)
	frame, err = ecfr.PointFrameTo(buf)
	if err != nil {
		return err
	}*/
	frame, err = cf.framer.New(CommandFramerMaxDatagramsLen)
	if err != nil {
		return err
	}

	cf.currentFrame = frame
	cf.currentDgram = nil
	cf.currentCmds = nil
	cf.frameOpen = true
	cf.currentFrameLen = CommandFramerMaxDatagramsLen
	cf.currentFrameOffset = 0
	return nil
}

func (cf *CommandFramer) Cycle() error {
	if cf.currentFrame != nil && len(cf.currentFrame.Datagrams) > 0 {
		cf.finishFrame()
	}

	/*for i, of := range cf.frameQueue {
		fr := of.frame
		frbuf, err := fr.Commit()
		if err != nil {
			return err
		}

		var f ecfr.Frame
		_, err = f.Overlay(frbuf)
		if err != nil {
			return err
		}

		fmt.Printf("frameQueue entry %d len %d\n", i, len(frbuf))
		for _, dgram := range f.Datagrams {
			fmt.Println("  ", dgram.Summary())
		}
		fmt.Println()
	}*/

	var err error
	cf.inFrameQueue, err = cf.framer.Cycle()
	if err != nil {
		return err
	}

	//for i, fr := range cf.inFrameQueue {
	/*frbuf, err := fr.Commit()
	if err != nil {
		return err
	}

	var f ecfr.Frame
	_, err = f.Overlay(frbuf)
	if err != nil {
		return err
	}*/

	/*fmt.Printf("inFrameQueue entry %d len %d\n", i, fr.ByteLen())
	for _, dgram := range fr.Datagrams {
		fmt.Println("  ", dgram.Summary())
	}
	fmt.Println()*/
	//}

	oi := 0
	for _, infr := range cf.inFrameQueue {
		if oi == len(cf.frameQueue) {
			// no more outgoing frames to scan
			break
		}

		for i := oi; i < len(cf.frameQueue); i++ {
			// is this outgoing frame a match for the incoming frame?
			ofr := cf.frameQueue[i].frame
			if infr.Header.FrameLength() != ofr.Header.FrameLength() {
				continue
			}

			if len(infr.Datagrams) == 0 || len(ofr.Datagrams) == 0 {
				continue
			}

			if len(infr.Datagrams) != len(ofr.Datagrams) {
				continue
			}

			if infr.Datagrams[0].Index != ofr.Datagrams[0].Index {
				continue
			}

			// TODO: more criteria
			for j, ocmd := range cf.frameQueue[i].cmds {
				odgram := ocmd.DatagramOut
				indgram := infr.Datagrams[j]

				if odgram.Command != indgram.Command {
					continue
				}

				if odgram.DataLength() != indgram.DataLength() {
					continue
				}

				ocmd.DatagramIn = indgram
				ocmd.Arrived = true
				ocmd.Overlayed = true
				ocmd.Error = nil
			}

			// update search start index
			oi = i
		}
	}

	cf.frameQueue = nil
	cf.inFrameQueue = nil

	return nil
}

func (cf *CommandFramer) Close() error {
	return nil
}

type Framer interface {
	New(maxdatalen int) (*ecfr.Frame, error)
	Cycle() ([]*ecfr.Frame, error)
}
