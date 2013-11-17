package sim

import (
	"github.com/distributed/ecat/ecad"
	"github.com/distributed/ecat/ecfr"
)

const (
	regAreaLength = 0x1000
)

type FrameProcessor interface {
	ProcessFrame(*ecfr.Frame) *ecfr.Frame
}

type L2Slave struct {
	BackingMemory [1 << 16]byte

	registerShadow          [regAreaLength]byte
	registerShadowWriteMask [regAreaLength]bool

	regMappings []MMapping

	ALStatusControl *ALStatusControl
	EEPROM          *L2EEPROM
}

func NewL2Slave() *L2Slave {
	s := &L2Slave{}

	// ET1100 signature
	copy(s.BackingMemory[:0x10], []byte{0x11, 0x00, 0x02, 0x00, 0x08, 0x08, 0x08, 0x0b, 0xfc})

	s.ALStatusControl = NewALStatusControl()
	s.regMappings = append(s.regMappings, DevMapping{ecad.ALControl, 0x02, s.ALStatusControl.ControlReg()})
	s.regMappings = append(s.regMappings, DevMapping{ecad.ALStatus, 0x06, s.ALStatusControl.StatusReg()})

	s.EEPROM = NewL2EEPROM()
	s.regMappings = append(s.regMappings, DevMapping{ecad.ESIEEPROMInterface, 0x10, s.EEPROM.Reg()})

	return s
}

// returns true if interaction happened
func (s L2Slave) llread8p(addr uint16, dp *uint8) bool {
	if addr < regAreaLength {
		// register access
		m := s.addrToMapping(addr)
		if m != nil {
			return m.Device().Read(addr-m.Start(), dp)
		}
	}

	*dp = s.BackingMemory[addr]
	return true
}

// returns true if interaction happened.
func (s *L2Slave) llwrite8(addr uint16, d uint8) bool {
	if addr < regAreaLength {
		s.registerShadow[addr] = d
		s.registerShadowWriteMask[addr] = true

		// TODO: need to consult regs if writing is OK
		m := s.addrToMapping(addr)
		if m != nil {
			return m.Device().WriteInteract(addr - m.Start())
		}
	}

	// no support for sync managers so far
	s.BackingMemory[addr] = d
	return true
}

func (s *L2Slave) addrToMapping(addr uint16) MMapping {
	for _, m := range s.regMappings {
		if addr >= m.Start() && addr < (m.Start()+m.Length()) {
			return m
		}
	}

	return nil
}

func (s *L2Slave) ProcessFrame(infr *ecfr.Frame) (ofr *ecfr.Frame) {
	ofr = infr

	for _, dg := range infr.Datagrams {
		// TODO: should ecfr.Frame contain a DatagramAddress instead of Addr32?
		if s.isPhysicalAddr(dg.Command, dg.Addr32) {
			dga := ecfr.DatagramAddressFromCommand(dg.Addr32, dg.Command)
			physaddressed := s.isPhysicallyAdressed(dga)
			dga.IncrementSlaveAddr()
			dg.Addr32 = dga.Addr32()
			if !physaddressed {
				continue
			}

			readUnmasked := true
			if dg.Command.DoesRead() {
				physbase := dga.Offset()
				for i := uint16(0); i < dg.DataLength(); i++ {
					//di := dg.Data()[i]
					readUnmasked = s.llread8p(physbase+i, &(dg.Data()[i])) && readUnmasked
					//do := dg.Data()[i]
					//fmt.Printf("llread8p di %02x -> do %02x  @  %p\n", di, do, &(dg.Data()[i]))
				}
			}

			writeUnmasked := true
			if dg.Command.DoesWrite() {
				physbase := dga.Offset()
				for i := uint16(0); i < dg.DataLength(); i++ {
					writeUnmasked = s.llwrite8(physbase+i, dg.Data()[i]) && writeUnmasked
				}
			}

			// working counter update logic
			if dg.Command.DoesRead() && dg.Command.DoesWrite() {
				// TODO: RW/ARMW update logic
			} else if dg.Command.DoesRead() {
				if readUnmasked {
					dg.WorkingCounter++
				}
			} else if dg.Command.DoesWrite() {
				if writeUnmasked {
					dg.WorkingCounter++
				}
			}
		}
		// no support for logical addresses
	}

	// latch register shadow into registers
	s.latchRegs()
	// frame is processed

	return
}

func (s *L2Slave) latchRegs() {
	for _, m := range s.regMappings {
		start := m.Start()
		end := start + m.Length()
		m.Device().Latch(s.registerShadow[start:end],
			s.registerShadowWriteMask[start:end])
	}
}

func (s *L2Slave) isPhysicalAddr(ct ecfr.CommandType, addr32 uint32) bool {
	dga := ecfr.DatagramAddressFromCommand(addr32, ct)
	return dga.IsPhysical()
}

func (s *L2Slave) isPhysicallyAdressed(addr ecfr.DatagramAddress) bool {
	if addr.Type() == ecfr.Broadcast {
		return true
	}

	if addr.Type() == ecfr.Positional {
		return addr.PositionOrAddress() == 0
	}

	if addr.Type() == ecfr.Fixed {
		// TODO: station address reg
		return false
	}

	return false
}

func NewALStatusControl() *ALStatusControl {
	return &ALStatusControl{Store: 0x0011}
}

type ALStatusControl struct {
	Store uint16
}

func (a *ALStatusControl) IsECATWritable() bool {
	return true
}

func (a *ALStatusControl) InError() bool {
	return (a.Store & 0x10) != 0
}

func (a *ALStatusControl) SetError(seterr bool) {
	if seterr {
		a.Store |= 0x10
	} else {
		a.Store &^= 0x10
	}
}

type ALControl struct{ *ALStatusControl }

func (sc *ALStatusControl) ControlReg() ALControl { return ALControl{sc} }

func (c ALControl) Read(offs uint16, dp *uint8) bool {
	switch offs {
	case 0:
		*dp = uint8(c.Store)
	case 1:
		*dp = uint8(c.Store >> 8)
	default:
		panic("invalid mapping for ALControl exceeds possible length")
	}

	return true
}

func (c ALControl) WriteInteract(offs uint16) bool {
	return c.IsECATWritable()
}

func (c ALControl) Latch(shadow []byte, shadowWriteMask []bool) {
	if shadowWriteMask[0] {
		if (c.InError() && (shadow[0]&0x10) != 0) || !c.InError() {
			c.Store &^= 0x1f
			c.Store |= uint16(shadow[0] & 0x0f)
		}
	}
}

type ALStatus struct{ *ALStatusControl }

func (sc *ALStatusControl) StatusReg() ALStatus { return ALStatus{sc} }

func (s ALStatus) Read(offs uint16, dp *uint8) bool {
	//fmt.Printf("AL Status Read offs %d, dp %p\n", offs, dp)
	switch offs {
	case 0:
		*dp = uint8(s.Store)
		//fmt.Printf("read 0, *dp %#02x\n", *dp)
	case 1:
		*dp = uint8(s.Store >> 8)
		//fmt.Printf("read 1, AL Store %04x\n", s.Store)
	default:
		*dp = 0x00
	}
	return true
}

func (s ALStatus) WriteInteract(offs uint16) bool {
	return false
}

func (s ALStatus) Latch(shadow []byte, shadowWriteMask []bool) {}
