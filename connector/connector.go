package main

import (
	"log"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// #define CN_IDX_PROC                     0x1
	cnIdxProc uint32 = 0x1
	// #define CN_VAL_PROC                     0x1
	cnValProc uint32 = 0x1

	/*
	 * Userspace sends this enum to register with the kernel that it is listening
	 * for events on the connector.
	 */
	PROC_CN_MCAST_LISTEN = 1
	PROC_CN_MCAST_IGNORE = 2
)

var (
	seq uint32 = 0

	sa = &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: cnIdxProc,
		Pid:    uint32(os.Getpid()),
	}
)

// map c struct cb_id defined in connector.h
type cbID struct {
	idx uint32
	val uint32
}

// map c struct cn_msg defined in connector.h
type cnMsg struct {
	id    cbID
	seq   uint32
	ack   uint32
	len   uint16
	flags uint16
	// data  int8 该处为　proc msg 地址
}

type nlSocket struct {
	fd     int
	binded bool
}

func (nl *nlSocket) Connect() error {
	err := unix.Bind(nl.fd, sa)
	nl.binded = (err == nil)
	return err
}

// func (nl *nlSocket) enableMonitor(en bool) {
// 	cnmsg := cnMsg{
// 		id: cbID{cnIdxProc, cnValProc},
// 	}
// 	nlh := unix.NlMsghdr{
// 		Len:   uint32(unix.NLMSG_HDRLEN) + (uint32)(unsafe.Sizeof(cnmsg)),
// 		Type:  unix.NLMSG_DONE,
// 		Flags: 0, // /usr/include/linux/netlink.h
// 		Pid:   uint32(os.Getpid()),
// 		Seq:   seq,
// 	}
// 	unix.Sendto(nl.fd, []byte{}, 0, sa)
// }

func getSocket() (*nlSocket, error) {
	sock, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_CONNECTOR)
	if err != nil {
		return nil, err
	}
	return &nlSocket{sock, false}, nil
}

func main() {
	nl, err := getSocket()
	if err != nil {
		log.Fatalf("socket: %v\n", err)
	}
	if err := nl.Connect(); err != nil {
		log.Fatalf("bind: %v\n", err)
	}

	cnmsg := cnMsg{
		id:  cbID{cnIdxProc, cnValProc},
		seq: seq,
		ack: 0,
		len: 1,
	}
	nlh := unix.NlMsghdr{
		Len:   uint32(unix.NLMSG_HDRLEN) + (uint32)(unsafe.Sizeof(cnmsg)),
		Type:  unix.NLMSG_DONE,
		Flags: 0, // /usr/include/linux/netlink.h
		Pid:   uint32(os.Getpid()),
		Seq:   seq,
	}
	unix.Sendto(nl.fd, []byte{}, 0, sa)
}
