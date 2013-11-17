package sim

type MMDevice interface {
	Read(offs uint16, dp *uint8) bool
	WriteInteract(offs uint16) bool
	Latch(shadow []byte, shadowWriteMask []bool)
}

type MMapping interface {
	Start() uint16
	Length() uint16
	Device() MMDevice
}

type DevMapping struct {
	StartAddr   uint16
	LengthField uint16
	DeviceField MMDevice
}

func (d DevMapping) Start() uint16    { return d.StartAddr }
func (d DevMapping) Length() uint16   { return d.LengthField }
func (d DevMapping) Device() MMDevice { return d.DeviceField }
