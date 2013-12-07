// +build !darwin

package udp

func errorMask(err error) error {
	return err
}
