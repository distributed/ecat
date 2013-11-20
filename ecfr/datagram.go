package ecfr

import (
	"bytes"
	"errors"
	"fmt"
)

type Datagram struct {
	DatagramHeader
	data           []byte
	WorkingCounter uint16
	buffer         []byte
}

func (dg *Datagram) Overlay(d []byte) (b []byte, err error) {
	//fmt.Printf("overlaying datagram over %s\n", spew.Sdump(d))
	b, err = dg.DatagramHeader.Overlay(d)
	if err != nil {
		return
	}

	if len(b) < int(dg.DataLength()) {
		err = fmt.Errorf("overlaying ecat dgram: need %d bytes of data, have %d", dg.DataLength(), len(b))
		return
	}

	dg.data = b[:dg.DataLength()]
	b = b[dg.DataLength():]

	if len(b) < 2 {
		err = fmt.Errorf("overlaying ecat dgram: need 2 bytes for working counter, got %d", len(b))
		return
	}

	// guarded by condition above
	dg.WorkingCounter, b = getUint16(b)

	dg.buffer = d
	return
}

func (dg *Datagram) Commit() (d []byte, err error) {
	_, err = dg.DatagramHeader.Commit()
	if err != nil {

	}

	// the data is already committed

	wcoffs := datagramHeaderLength + len(dg.data)
	if len(dg.buffer) < (wcoffs + 2) {
		err = fmt.Errorf("cannot commit data: buffer too short, need %d bytes, have %d", wcoffs+2, len(dg.buffer))
		return
	}

	putUint16(dg.buffer[wcoffs:wcoffs+2], dg.WorkingCounter)

	d = dg.buffer[:wcoffs+2]

	return
}

func (dg *Datagram) ByteLen() int {
	return DatagramOverheadLength + len(dg.Data())
}

type DatagramHeader struct {
	Command   CommandType
	Index     uint8
	Addr32    uint32
	LenWord   uint16
	Interrupt uint16
	buffer    []byte
}

const (
	datagramHeaderByteLen = 10
)

func (dh *DatagramHeader) Overlay(d []byte) (b []byte, err error) {
	b = d
	if len(b) < datagramHeaderByteLen {
		err = fmt.Errorf("need %d bytes for dgram header, have %d", datagramHeaderByteLen, len(b))
		return
	}

	var c8 uint8
	c8, b = getUint8(b)
	dh.Command = CommandType(c8)
	dh.Index, b = getUint8(b)
	dh.Addr32, b = getUint32(b)
	dh.LenWord, b = getUint16(b)
	dh.Interrupt, b = getUint16(b)

	dh.buffer = d

	return
}

func (dh *DatagramHeader) Commit() (d []byte, err error) {
	b := dh.buffer

	b = putUint8(b, uint8(dh.Command))
	b = putUint8(b, dh.Index)
	b = putUint32(b, dh.Addr32)
	b = putUint16(b, dh.LenWord)
	b = putUint16(b, dh.Interrupt)

	d = dh.buffer[:len(b)]
	return
}

func (dh *DatagramHeader) SlaveAddr() uint16 {
	return uint16(dh.Addr32)
}

func (dh *DatagramHeader) OffsetAddr() uint16 {
	return uint16(dh.Addr32 >> 16)
}

func (dh *DatagramHeader) LogicalAddr() uint32 {
	return dh.Addr32
}

func (dh *DatagramHeader) DataLength() uint16 {
	return dh.LenWord & ((1 << 11) - 1)
}

func (dh *DatagramHeader) Roundtrip() bool {
	return (dh.LenWord & (1 << roundtripBit)) != 0
}

func (dh *DatagramHeader) Last() bool {
	return (dh.LenWord & (1 << lastindicatorBit)) == 0
}

func (dh *DatagramHeader) SetLast(last bool) {
	if last {
		dh.LenWord &^= (1 << lastindicatorBit)
	} else {
		dh.LenWord |= (1 << lastindicatorBit)
	}
}

func (dg *Datagram) Summary() string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "%-4s x%02x %04x %04x len %03x wc % 2d IRQ %04x ", dg.Command.String(),
		dg.Index,
		dg.SlaveAddr(),
		dg.OffsetAddr(),
		dg.DataLength(),
		dg.WorkingCounter,
		dg.Interrupt)

	cutoffstr := ""
	data := dg.Data()
	if len(data) > 8 {
		data = data[:8]
		cutoffstr = " ..."
	}

	fmt.Fprintf(buf, "> % x%s", data, cutoffstr)

	switch len(dg.Data()) {
	case 2:
		fmt.Fprintf(buf, " (%04x)", xgetUint16(dg.Data()))
	case 4:
		fmt.Fprintf(buf, " (%04x %04x | %08x)", xgetUint16(dg.Data()), xgetUint16(dg.Data()[2:]), xgetUint32(dg.Data()))
	}

	return buf.String()
}

func PointDatagramHeaderTo(d []byte) (dh DatagramHeader, err error) {
	if len(d) < datagramHeaderLength {
		err = fmt.Errorf("datagram header needs %d bytes, only have %d", datagramHeaderLength, len(d))
		return
	}

	dh.buffer = d[0:datagramHeaderLength]
	return
}

func (dg *Datagram) Data() []byte {
	return dg.data
}

func (dg *Datagram) SetDataLen(ndl int) error {
	nl := DatagramOverheadLength + ndl
	if (cap(dg.buffer)) < nl {
		return fmt.Errorf("datagram with new size needs %d bytes of space, only %d in buffer", nl, cap(dg.buffer))
	}

	if ndl > datagramMaxDataLength {
		return errors.New("new data length exceeds maximum datagram data length")
	}

	dg.data = dg.data[0:ndl]
	dg.buffer = dg.buffer[0:nl]

	dg.LenWord &^= datagramDataLengthMask
	dg.LenWord |= (uint16(ndl) & datagramDataLengthMask)

	return nil
}

func PointDatagramTo(d []byte) (dg Datagram, err error) {
	if len(d) < DatagramOverheadLength {
		err = errors.New("byte slice too short to be pointed to by a datagram")
		return
	}

	dg = Datagram{
		buffer: d,
		data:   d[datagramHeaderLength:datagramHeaderLength],
	}
	dg.DatagramHeader, err = PointDatagramHeaderTo(d)

	return
}

const (
	roundtripBit     = 14
	lastindicatorBit = 15

	datagramHeaderLength   = 10
	datagramFooterLength   = 2
	DatagramOverheadLength = datagramHeaderLength + datagramFooterLength

	datagramDataLengthMask = (1 << 12) - 1
	datagramMaxDataLength  = datagramDataLengthMask
)

type CommandType uint8

func (ct CommandType) String() string {
	if cts, ok := commandTypeName[ct]; ok {
		return cts
	}
	return fmt.Sprintf("CommandType(%d)", uint(ct))
}

type DatagramAddressType uint

const (
	UninitializedDatagramAddressType DatagramAddressType = 0
	Positional                       DatagramAddressType = 1
	Fixed                            DatagramAddressType = 2
	Broadcast                        DatagramAddressType = 3
	Logical                          DatagramAddressType = 4
)

type DatagramAddress struct {
	addr uint32
	typ  DatagramAddressType
}

func (d DatagramAddress) String() string {
	b := bytes.NewBuffer(nil)
	switch d.Type() {
	case Positional:
		b.WriteByte('p')
	case Fixed:
		b.WriteByte('f')
	case Broadcast:
		b.WriteByte('b')
	case Logical:
		b.WriteByte('l')
	default:
		b.WriteByte('U')
	}

	switch d.Type() {
	case Logical:
		fmt.Fprintf(b, "%08x", d.addr)
	default:
		fmt.Fprintf(b, "%04x.%04x", uint16(d.addr), uint16(d.addr>>16))
	}

	return b.String()
}

func (d DatagramAddress) Addr32() uint32 {
	return d.addr
}

func (d DatagramAddress) Type() DatagramAddressType {
	return d.typ
}

func DatagramAddressFromCommand(addr32 uint32, ct CommandType) DatagramAddress {
	typ := UninitializedDatagramAddressType
	var ok bool
	if typ, ok = datagramAddressByOperation[ct]; ok {
		return DatagramAddress{addr32, typ}
	}
	return DatagramAddress{addr32, typ}
}

func (d DatagramAddress) Offset() uint16 {
	return uint16(d.Addr32() >> 16)
}

func (d *DatagramAddress) SetOffset(offs uint16) {
	d.addr &^= 0xffff0000
	d.addr |= uint32(offs) << 16
}

func (d DatagramAddress) PositionOrAddress() uint16 {
	return uint16(d.Addr32())
}

func (d *DatagramAddress) IncrementSlaveAddr() {
	d.addr = (d.addr & 0xffff0000) | ((d.addr + 1) & 0x0000ffff)
}

func (d *DatagramAddress) IsPhysical() bool {
	switch d.Type() {
	case Positional:
	case Fixed:
	case Broadcast:
	default:
		return false
	}
	return true
}

func PositionalAddr(position int16, offset uint16) DatagramAddress {
	return DatagramAddress{uint32(uint16(position)) | uint32(offset)<<16, Positional}
}

func FixedAddr(stationaddr uint16, offset uint16) DatagramAddress {
	return DatagramAddress{uint32(stationaddr) | uint32(offset)<<16, Fixed}
}

var datagramAddressByOperation = map[CommandType]DatagramAddressType{
	NOP:  UninitializedDatagramAddressType,
	APRD: Positional,
	APWR: Positional,
	APRW: Positional,
	FPRD: Fixed,
	FPWR: Fixed,
	FPRW: Fixed,
	BRD:  Broadcast,
	BWR:  Broadcast,
	BRW:  Broadcast,
	LRD:  Logical,
	LWR:  Logical,
	LRW:  Logical,
	ARMW: Positional,
	FRMW: Positional,
}

const (
	NOP  CommandType = 0
	APRD CommandType = 1
	APWR CommandType = 2
	APRW CommandType = 3
	FPRD CommandType = 4
	FPWR CommandType = 5
	FPRW CommandType = 6
	BRD  CommandType = 7
	BWR  CommandType = 8
	BRW  CommandType = 9
	LRD  CommandType = 10
	LWR  CommandType = 11
	LRW  CommandType = 12
	ARMW CommandType = 13
	FRMW CommandType = 14
)

var commandTypeName = map[CommandType]string{
	NOP:  "NOP",
	APRD: "APRD",
	APWR: "APWR",
	APRW: "APRW",
	FPRD: "FPRD",
	FPWR: "FPWR",
	FPRW: "FPRW",
	BRD:  "BRD",
	BWR:  "BWR",
	BRW:  "BRW",
	LRD:  "LRD",
	LWR:  "LWR",
	LRW:  "LRW",
	ARMW: "ARMW",
	FRMW: "FRMW",
}

func (ct CommandType) DoesRead() bool {
	switch ct {
	case APRD:
	case APRW:
	case FPRD:
	case FPRW:
	case BRD:
	case BRW:
	case LRD:
	case LRW:
	case ARMW:
	case FRMW:
	default:
		return false
	}
	return true
}

func (ct CommandType) DoesWrite() bool {
	switch ct {
	case APWR:
	case APRW:
	case FPWR:
	case FPRW:
	case BWR:
	case BRW:
	case LWR:
	case LRW:
	case ARMW:
	case FRMW:
	default:
		return false
	}
	return true
}
