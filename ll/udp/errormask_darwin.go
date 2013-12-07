// +build darwin

package udp

import (
	"net"
	"syscall"
)

// on darwin we filter out "can't assign requested address" errors. these happen
// when there is no link or when the interface has not yet been properly
// configured with IP addresses.

func errorMask(err error) error {
	if oe, ok := err.(*net.OpError); ok {
		if se, ok := oe.Err.(syscall.Errno); ok {
			if se == syscall.EADDRNOTAVAIL {
				return nil
			}
		}
	}
	return err
}
