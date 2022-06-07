package ecee

import (
	"errors"
	"fmt"
	"github.com/distributed/ecat/ecad"
	"github.com/distributed/ecat/ecfr"
	"github.com/distributed/ecat/ecmd"
	"time"
)

type blindEEPROM struct {
	addr         ecfr.DatagramAddress
	commander    ecmd.Commander
	readCommand  ecfr.CommandType
	writeCommand ecfr.CommandType
	closed       bool
}

type EEPROM interface {
	ReadWord(addr uint32) (word uint16, err error)
	WriteWord(addr uint32, word uint16) (err error)
	Close() error
}

func New(commander ecmd.Commander, addr ecfr.DatagramAddress) (EEPROM, error) {
	ee := &blindEEPROM{
		addr:      addr,
		commander: commander,
	}

	err := ee.waitForIdle(0)
	if err != nil {
		return nil, err
	}

	return ee, nil
}

func (ee *blindEEPROM) waitForIdle(timeout time.Duration) error {
	if timeout == 0 {
		timeout = 250 * time.Millisecond
	}

	tot := time.Now().Add(timeout)

	for {
		addr := ee.addr
		addr.SetOffset(ecad.EEPROMControlStatus)
		rb, err := ecmd.ExecuteRead(ee.commander, addr, 2, 1)
		if err != nil {
			return err
		}

		if rb[1]&0x80 == 0 {
			return nil
		}

		if time.Now().After(tot) {

		}
	}
}

func (ee *blindEEPROM) ReadWord(addr uint32) (word uint16, err error) {
	if ee.closed {
		err = errors.New("ecee eeprom is already closed")
		return
	}

	err = ee.waitForIdle(0)
	if err != nil {
		return
	}

	dgaddr := ee.addr

	// write EEPROM address to ESC
	dgaddr.SetOffset(ecad.EEPROMAddress)
	wb := make([]byte, 4)
	wb[0] = uint8(addr)
	wb[1] = uint8(addr >> 8)
	wb[2] = uint8(addr >> 16)
	wb[3] = uint8(addr >> 24)
	err = ecmd.ExecuteWrite(ee.commander, dgaddr, wb, 1)
	if err != nil {
		return
	}

	// write "read command"
	dgaddr.SetOffset(ecad.EEPROMControlStatus)
	wb = []byte{0x00, 0x01} // read command
	err = ecmd.ExecuteWrite(ee.commander, dgaddr, wb, 1)
	if err != nil {
		return
	}

	err = ee.waitForIdle(0)
	if err != nil {
		return
	}

	// check error bits
	dgaddr.SetOffset(ecad.EEPROMControlStatus)
	var rb []byte
	rb, err = ecmd.ExecuteRead(ee.commander, dgaddr, 2, 1)
	if err != nil {
		return
	}

	if rb[1]&0xE0 != 0x00 {
		err = fmt.Errorf("EEPROM status word bits indicate error, bytes are % x\n", rb)
		return
	}

	dgaddr.SetOffset(ecad.EEPROMData)
	rb, err = ecmd.ExecuteRead(ee.commander, dgaddr, 4, 1)
	if err != nil {
		return
	}

	//fmt.Printf("EEPROM read 4 bytes: % x\n", rb)

	word = uint16(rb[0]) | uint16(rb[1])<<8
	return
}

func (ee *blindEEPROM) WriteWord(addr uint32, word uint16) (err error) {
	if ee.closed {
		err = errors.New("ecee eeprom is already closed")
		return
	}

	err = ee.waitForIdle(0)
	if err != nil {
		return
	}

	dgaddr := ee.addr

	// write EEPROM address to ESC
	dgaddr.SetOffset(ecad.EEPROMAddress)
	wb := make([]byte, 4)
	wb[0] = uint8(addr)
	wb[1] = uint8(addr >> 8)
	wb[2] = uint8(addr >> 16)
	wb[3] = uint8(addr >> 24)
	err = ecmd.ExecuteWrite(ee.commander, dgaddr, wb, 1)
	if err != nil {
		return
	}
	
	// write data
	dgaddr.SetOffset(ecad.EEPROMData)
	wb = []byte{uint8(word), uint8(word >> 8)}
	err = ecmd.ExecuteWrite(ee.commander, dgaddr, wb, 1)
	if err != nil {
		return
	}

	// write "write command"
	dgaddr.SetOffset(ecad.EEPROMControlStatus)
	wb = []byte{0x01, 0x02} // write command
	err = ecmd.ExecuteWrite(ee.commander, dgaddr, wb, 1)
	if err != nil {
		return
	}

	err = ee.waitForIdle(0)
	if err != nil {
		return
	}

	// check error bits
	dgaddr.SetOffset(ecad.EEPROMControlStatus)
	var rb []byte
	rb, err = ecmd.ExecuteRead(ee.commander, dgaddr, 2, 1)
	if err != nil {
		return
	}

	if rb[1]&0xE0 != 0x00 {
		err = fmt.Errorf("EEPROM status word bits indicate error, bytes are % x\n", rb)
		return
	}

	dgaddr.SetOffset(ecad.EEPROMData)
	rb, err = ecmd.ExecuteRead(ee.commander, dgaddr, 4, 1)
	if err != nil {
		return
	}
	
	return
}

func (ee *blindEEPROM) Close() error {
	ee.closed = true
	return nil
}
