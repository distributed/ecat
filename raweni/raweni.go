package raweni

import (
	"code.google.com/p/go-charset/charset"
	"encoding/xml"
	"github.com/davecgh/go-spew/spew"
	"io"
	"os"
	"strconv"
	"strings"
)

import _ "code.google.com/p/go-charset/data"

func ReadEtherCATInfoFromFile(filename string) (eci EtherCATInfo, err error) {
	var r io.Reader
	r, err = os.Open(filename)
	if err != nil {
		return
	}

	return ReadEtherCATInfo(r)
}

func ReadEtherCATInfo(r io.Reader) (eci EtherCATInfo, err error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReader

	err = dec.Decode(&eci)
	if err != nil {
		spew.Dump(err)
		return
	}

	return
}

type EtherCATInfo struct {
	Vendor       Vendor
	Descriptions Descriptions
}

type Vendor struct {
	Id   uint32
	Name string
}

type Descriptions struct {
	Groups  []Group  `xml:"Groups>Group"`
	Devices []Device `xml:"Devices>Device"`
}

type Group struct {
	Type  string
	Names []GroupName `xml:"Name"`
}

type GroupName struct {
	LcIdentifiedName
}

type LcIdentifiedName struct {
	String string `xml:",chardata"`
	LcId   uint   `xml:",attr"`
}

type Device struct {
	Type   DeviceType
	Names  []LcIdentifiedName `xml:"Name"`
	Sms    []Sm               `xml:"Sm"`
	Eeprom Eeprom
}

type DeviceType struct {
	Name           string `xml:",chardata"`
	ProductCodeRaw string `xml:"ProductCode,attr"`
	RevisionNoRaw  string `xml:"RevisionNo,attr"`
}

func (d DeviceType) ProductCode() uint32 {
	return uint32(bh2i(d.ProductCodeRaw))
}

func (d DeviceType) RevisionNo() uint32 {
	return uint32(bh2i(d.RevisionNoRaw))
}

type Sm struct {
	Name                          string `xml:",chardata"`
	MinSize, MaxSize, DefaultSize uint   `xml:",attr"`
	StartAddressRaw               string `xml:"StartAddress,attr"`
	ControlByteRaw                string `xml:"ControlByte,attr"`
}

func (s Sm) StartAddress() uint16 {
	return uint16(bh2i(s.StartAddressRaw))
}

func (s Sm) ControlByte() uint8 {
	return uint8(bh2i(s.ControlByteRaw))
}

// beckhoff hex string to integer, 0 on failure
func bh2i(s string) uint64 {
	var (
		n   uint64
		err error
	)

	if strings.HasPrefix(s, "#x") {
		// as s has 2 byte prefix, indexing is OK
		n, err = strconv.ParseUint(s[2:], 16, 64)
	} else {
		n, err = strconv.ParseUint(s, 10, 64)
	}

	if err != nil {
		return 0
	}

	return n
}

type Eeprom struct {
	ByteSize      uint
	ConfigDataRaw string `xml:"ConfigData"`
}
