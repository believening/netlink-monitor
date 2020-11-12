package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	MaxBufferSize = 2048

	CN_IDX_PROC = 0x1
	CN_VAL_PROC = 0x1

	procCnMCastListen = 1
	procCnMCastIgnore = 2
)

var nextSeqNr uint32
var seq uint32

type socket struct {
	sd int32
}

func (s *socket) Close() {
	fd := int(atomic.SwapInt32(&s.sd, -1))
	unix.Close(fd)
}

func (s *socket) Fd() int {
	return int(s.sd)
}

func getSocket() (*socket, error) {
	sd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_CONNECTOR)
	if err != nil {
		return nil, err
	}
	return &socket{
		sd: int32(sd),
	}, nil
}

func initNladdr() (sa, da *syscall.SockaddrNetlink) {
	sa = &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: CN_IDX_PROC,
		Pid:    uint32(syscall.Getpid()),
	}
	da = &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
	}
	return
}

type nlMsgHdr struct {
	len uint32
	typ uint16
	flg uint16
	seq uint32
	pid uint32
}

type cbID struct {
	Idx uint32
	Val uint32
}

type cnMsg struct {
	ID    cbID
	Seq   uint32
	Ack   uint32
	Len   uint16
	Flags uint16
}

const nlMsgHdrSize = 0x10

func (h *nlMsgHdr) Marshal() []byte {
	b := make([]byte, nlMsgHdrSize)
	hdr := (*(*[nlMsgHdrSize]byte)(unsafe.Pointer(h)))[:]
	copy(b[0:nlMsgHdrSize], hdr)
	return b
}

func getSendMsg() []byte {
	seq++
	d := cnMsg{
		ID: cbID{
			Idx: CN_IDX_PROC, Val: CN_VAL_PROC,
		},
		Len: uint16(binary.Size(procCnMCastListen)),
	}
	h := nlMsgHdr{
		len: nlMsgHdrSize + uint32(binary.Size(d)+binary.Size(procCnMCastListen)),
		typ: syscall.NLMSG_DONE,
		flg: 0,
		seq: seq,
		pid: uint32(syscall.Getpid()),
	}
	buf := bytes.NewBuffer(make([]byte, 0, h.len))
	binary.Write(buf, binary.LittleEndian, &h)
	binary.Write(buf, binary.LittleEndian, &d)
	binary.Write(buf, binary.LittleEndian, procCnMCastListen)
	return buf.Bytes()
}

func main() {
	// get scoket
	sd, err := getSocket()
	if err != nil {
		log.Fatal("socket: ", err)
	}
	log.Println(sd.Fd())
	defer sd.Close()

	// init source and dest nelink addr
	sa, da := initNladdr()

	// bind
	if err = syscall.Bind(sd.Fd(), sa); err != nil {
		log.Fatalln("bind: ", err)
	}

	// send
	if err := syscall.Sendto(sd.Fd(), getSendMsg(), 0, da); err != nil {
		log.Fatalln("sendto: ", err)
	}

	buf := make([]byte, syscall.Getpagesize())
	for {
		log.Println("receiving...")
		nr, _, err := syscall.Recvfrom(sd.Fd(), buf[:], 0)
		if err != nil {
			log.Fatal("recvemsg: ", err)
		}
		msgs, err := syscall.ParseNetlinkMessage(buf[:nr])
		if err != nil {
			log.Fatalln("ParseNetlinkMessage:", err)

		}
		for _, msg := range msgs {
			if msg.Header.Type == syscall.NLMSG_DONE {
				readBuf := bytes.NewBuffer(msg.Data)
				fmt.Println(readBuf.String())
			}
		}
	}
}
