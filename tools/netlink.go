package tools

import "syscall"

func NetlinkSocket(proto int) (int, error) {
	return syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, proto)
}
