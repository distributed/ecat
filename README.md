ecat is an ethercat library for Go. It consists of the following parts:

ecfr provides framing of ethercat frames and datagrams.
ecmd provides execution of ethercat commands on top of link layer drivers,
sports a goroutine-safe command multiplexer and features a number of
convenience feature for command retries in case of frame loss or mismatching
working counters.
ecee provides (read only) access to ESC EEPROMs.
ecad contains a number of ESC register addresses.
ll contains link layer drivers of which there currently only one, using UDP
multicast.
raweni provides very raw access to ESI files. it's a misnomer.
sim contains rudimentary slave and bus simulation.
