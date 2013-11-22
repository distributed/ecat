package ecad

const (
	Type                 = 0x0000
	Revision             = 0x0001
	Build                = 0x0002
	FMMUsSupported       = 0x0004
	RAMSize              = 0x0006
	PortDescriptor       = 0x0007
	ESCFeaturesSupported = 0x0008

	ConfiguredStationAddress = 0x0010
	ConfiguredStationAlias   = 0x0012

	DLControl = 0x0100
	DLStatus  = 0x0110

	ALControl    = 0x0120
	ALStatus     = 0x0130
	ALStatusCode = 0x0134
	PDIControl   = 0x0140

	ECATEventMask = 0x0200

	ESIEEPROMInterface   = 0x0500
	EEPROMConfiguration  = 0x0500
	EEPROMPDIAccessState = 0x0501
	EEPROMControlStatus  = 0x0502
	EEPROMAddress        = 0x0504
	EEPROMData           = 0x0508

	FMMUBase = 0x0600

	SyncMangerBase                 = 0x0800
	SyncManagerChannelLen          = 0x08
	SyncManagerPhysStartAddrOffset = 0x00
	SyncManagerLengthOffset        = 0x02
	SyncManagerControlOffset       = 0x04
	SyncManagerStatusOffset        = 0x05
	SyncManagerActivateOffset      = 0x06
	SyncManagerPDIControlOffset    = 0x07
)
