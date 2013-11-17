package sim

type L2EEPROM struct {
	Array [8 * 1024]uint16

	Addr        uint32
	DataScratch [8]byte // already in wire encoding

	PDIControl         bool
	WriteEnable        bool
	ChecksumError      bool
	EENotLoaded        bool
	MissingAcknowledge bool
	ErrorWriteEnable   bool
	Busy               bool
}

func NewL2EEPROM() *L2EEPROM {
	ee := &L2EEPROM{}

	for i := 0; i < len(ee.Array); i++ {
		ee.Array[i] = 0xee00 + uint16(i)
	}

	return ee
}

func (ee *L2EEPROM) Reg() *L2EEPROMRegisterSet {
	return &L2EEPROMRegisterSet{ee}
}

type L2EEPROMRegisterSet struct{ *L2EEPROM }

func (ee *L2EEPROMRegisterSet) Read(offs uint16, dp *uint8) bool {
	switch offs {
	case 0:
		if ee.PDIControl {
			*dp = 0x01
		} else {
			*dp = 0x00
		}
	case 1:
		*dp = 0x00
	case 2:
		if ee.WriteEnable {
			*dp |= 0x01
		}
		*dp |= 0xc0 // 2 address bytes, support 8 bytes
	case 3:
		// lower 3 bits are command
		if ee.ChecksumError {
			*dp |= 1 << (11 - 8)
		}
		if ee.EENotLoaded {
			*dp |= 1 << (12 - 8)
		}
		if ee.MissingAcknowledge {
			*dp |= 1 << (13 - 8)
		}
		if ee.ErrorWriteEnable {
			*dp |= 1 << (14 - 8)
		}
		if ee.Busy {
			*dp |= 1 << (15 - 8)
		}
	case 4:
		*dp = uint8(ee.Addr)
	case 5:
		*dp = uint8(ee.Addr >> 8)
	case 6:
		*dp = uint8(ee.Addr >> 16)
	case 7:
		*dp = uint8(ee.Addr >> 24)
	default:
		if offs > 16 {
			panic("invalid use of ee reg area, read past end")
		}
		if offs >= 8 && offs < 16 {
			*dp = ee.DataScratch[offs-8]
		}
	}

	return true
}

func (ee *L2EEPROMRegisterSet) WriteInteract(offs uint16) bool {
	if offs == 2 || offs == 3 {
		return !ee.Busy
	}
	return true
}

func (ee L2EEPROMRegisterSet) Latch(shadow []byte, shadowWriteMask []bool) {
	for offs := 0; offs < len(shadow); offs++ {
		switch {
		case offs == 0:
			if shadowWriteMask[0] {
				if shadow[0]&0x01 != 0 {
					ee.PDIControl = true
				} else {
					ee.PDIControl = false
				}
			}
		case offs == 1:
			// pdi access state. we don't even fake that
		case offs == 2:
			if shadowWriteMask[2] {
				if shadow[2]&0x01 != 0 {
					ee.WriteEnable = true
				} else {
					ee.WriteEnable = false
				}
			}
		case offs == 3:
			if shadowWriteMask[3] {
				switch shadow[3] & 0x03 {
				case 0x00:
					ee.ChecksumError = false
					ee.EENotLoaded = false
					ee.MissingAcknowledge = false
					ee.ErrorWriteEnable = false
				case 0x01:
					// TODO: busy time/cycles
					ee.Busy = false
					ee.readIntoScratch()
				default:
					// write/reload not supported
				}
			}
		case offs == 4:
			if shadowWriteMask[4] {
				ee.Addr &^= 0xff
				ee.Addr |= uint32(shadow[4]) << 0
			}
		case offs == 5:
			if shadowWriteMask[5] {
				ee.Addr &^= 0xff00
				ee.Addr |= uint32(shadow[5]) << 8
			}
		case offs == 6:
			if shadowWriteMask[6] {
				ee.Addr &^= 0xff0000
				ee.Addr |= uint32(shadow[6]) << 16
			}
		case offs == 7:
			if shadowWriteMask[7] {
				ee.Addr &^= 0xff000000
				ee.Addr |= uint32(shadow[7]) << 32
			}

		case offs >= 8 && offs < 16:
			if shadowWriteMask[offs] {
				ee.DataScratch[offs-8] = shadow[offs]
			}
		}
	}
}

func (ee *L2EEPROM) readIntoScratch() {
	for i := 0; i < 4; i++ {
		w16 := ee.Array[(int(ee.Addr)+i)%len(ee.Array)]
		ee.DataScratch[i*2] = uint8(w16)
		ee.DataScratch[i*2+1] = uint8(w16 >> 8)
	}
}
