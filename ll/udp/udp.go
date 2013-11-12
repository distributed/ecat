package udp

import (
	"code.google.com/p/go.net/ipv4"
	"github.com/distributed/ecat/ecfr"
	"net"
	"time"
)

const (
	EthercatUDPPort = 0x88a4
)

const (
	udpReceiveBuflen = 1500
	maxDatagramsLen  = 1470
)

type UDPFramer struct {
	oframes []*ecfr.Frame

	sock      *net.UDPConn
	mcsock    *ipv4.PacketConn
	group     net.IP
	iface     *net.Interface
	laddr     *net.UDPAddr
	groupaddr *net.UDPAddr
	cycletime time.Duration
}

func NewUDPFramer(iface *net.Interface, group net.IP, cycletime time.Duration) (f *UDPFramer, err error) {
	f = &UDPFramer{}
	f.group = group
	f.iface = iface
	f.cycletime = cycletime

	f.laddr = &net.UDPAddr{net.IPv4(0, 0, 0, 0), EthercatUDPPort, ""}
	f.groupaddr = &net.UDPAddr{f.group, EthercatUDPPort, ""}

	f.sock, err = net.ListenUDP("udp4", f.laddr)
	if err != nil {
		return
	}

	f.mcsock = ipv4.NewPacketConn(f.sock)

	err = f.mcsock.SetMulticastInterface(f.iface)
	if err != nil {
		return
	}

	err = f.mcsock.JoinGroup(iface, &net.UDPAddr{IP: group})
	if err != nil {
		return
	}

	err = f.mcsock.SetMulticastLoopback(false)
	if err != nil {
		return
	}

	return
}

func (f *UDPFramer) New(maxdatalen int) (fr *ecfr.Frame, err error) {
	var vframe ecfr.Frame
	buf := make([]byte, maxDatagramsLen+ecfr.FrameOverheadLen)
	vframe, err = ecfr.PointFrameTo(buf)
	if err != nil {
		return
	}

	vframe.Header.SetType(1)

	fr = &vframe
	f.oframes = append(f.oframes, fr)
	return
}

func (f *UDPFramer) Cycle() (iframes []*ecfr.Frame, err error) {
	// TODO: send/receive concurrently to be independent of queue depth?

	var obytes []byte
	for _, oframe := range f.oframes {
		obytes, err = oframe.Commit()
		if err != nil {
			return
		}

		_, err = f.sock.WriteTo(obytes, f.groupaddr)
		if err != nil {
			return
		}
	}

	err = f.sock.SetDeadline(time.Now().Add(f.cycletime))

	rbuf := make([]byte, udpReceiveBuflen)
	for {
		var n int
		n, _, err = f.sock.ReadFromUDP(rbuf)
		if isTimeout(err) {
			err = nil
			break
		}
		if err != nil {
			return
		}

		var fr ecfr.Frame
		_, err = fr.Overlay(rbuf[0:n])
		if err != nil {
			// discard malformed frames
			continue
		}

		iframes = append(iframes, &fr)
		rbuf = make([]byte, udpReceiveBuflen)
	}

	return
}

func (f *UDPFramer) Close() error {
	if f.mcsock != nil {
		f.mcsock.Close()
	}
	if f.sock != nil {
		return f.Close()
	}
	return nil
}

type timeouter interface {
	Timeout() bool
}

func isTimeout(err error) bool {
	if t, ok := err.(timeouter); ok {
		return t.Timeout()
	}
	return false
}
