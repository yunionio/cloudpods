// +build !linux

package netutils2

func (n *SNetInterface) GetAddresses() [][]string {
	return nil
}
