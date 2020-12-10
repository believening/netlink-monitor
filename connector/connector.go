package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
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
	PROC_CN_MCAST_LISTEN uint32 = 1
	PROC_CN_MCAST_IGNORE uint32 = 2

	/* Use successive bits so the enums can be used to record
	 * sets of events as well
	 */
	PROC_EVENT_NONE   uint32 = 0x00000000
	PROC_EVENT_FORK   uint32 = 0x00000001
	PROC_EVENT_EXEC   uint32 = 0x00000002
	PROC_EVENT_UID    uint32 = 0x00000004
	PROC_EVENT_GID    uint32 = 0x00000040
	PROC_EVENT_SID    uint32 = 0x00000080
	PROC_EVENT_PTRACE uint32 = 0x00000100
	PROC_EVENT_COMM   uint32 = 0x00000200
	/* "next" should be 0x00000400 */
	/* "last" is the last process event: exit,
	 * while "next to last" is coredumping event */
	PROC_EVENT_COREDUMP uint32 = 0x40000000
	PROC_EVENT_EXIT     uint32 = 0x80000000

	CN_MSG_LEN     uint32 = 20
	NLC_MSG_LEN    uint32 = unix.NLMSG_HDRLEN + CN_MSG_LEN
	PROC_HDR_LEN   uint32 = 16
	PROC_EVENT_LEN uint32 = 40
)

var (
	order binary.ByteOrder

	seq uint32 = 0

	sa = &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: cnIdxProc,
		Pid:    uint32(os.Getpid()),
	}

	bufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, NLC_MSG_LEN+PROC_EVENT_LEN, NLC_MSG_LEN+PROC_EVENT_LEN)
		},
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
	// data  int8 该处为　proc msg 的首地址, 参考　map 实现
}

type nlConnectorMsg struct {
	header unix.NlMsghdr
	msg    cnMsg
}

// packaged together
type nlConnectorMsgPacked struct {
	header unix.NlMsghdr
	// map c struct cn_msg defined in connector.h
	cnMsg struct {
		// map c struct cb_id defined in connector.h
		cbID struct {
			idx uint32
			val uint32
		}
		seq   uint32
		ack   uint32
		len   uint16
		flags uint16
		// data  int8 该处为　proc msg 的首地址
	}
}

// map c struct proc_event defined in cn_proc.h
type procEventHeader struct {
	what        uint32
	cpu         uint32
	timestampns uint64
}

type pidAndTgid struct {
	pid, tgid uint32 // in c program language, sizeof(int) is 4
}

type ackProc struct {
	err uint32
}

type forkProc struct {
	parent, child pidAndTgid
}

type execProc struct {
	process pidAndTgid
}

type idProc struct {
	process pidAndTgid
	r, e    uint32
}

type sidProc struct {
	process pidAndTgid
}

type ptraceProc struct {
	process, tracer pidAndTgid
}

type commProc struct {
	process pidAndTgid
	comm    [16]byte
}

type coredumpProc struct {
	process, parent pidAndTgid
}

type exitProc struct {
	process             pidAndTgid
	exitCode, exitSinal uint32
	parent              pidAndTgid
}

func init() {
	x := 0x1001
	if *(*byte)(unsafe.Pointer(&x)) == 0x10 {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
}

type nlSocket struct {
	fd     int
	binded bool
}

func (nl *nlSocket) Close() error {
	nl.binded = false
	return unix.Close(nl.fd)
}

func (nl *nlSocket) Connect() error {
	err := unix.Bind(nl.fd, sa)
	nl.binded = (err == nil)
	return err
}

func (nl *nlSocket) enableMonitor(en bool) error {
	if !nl.binded {
		return fmt.Errorf("netlink not bind")
	}
	defer func() {
		atomic.AddUint32(&seq, 1)
	}()

	var op uint32
	if en {
		op = PROC_CN_MCAST_LISTEN
	} else {
		op = PROC_CN_MCAST_IGNORE
	}
	dataLen := binary.Size(op)
	msg := nlConnectorMsg{
		header: unix.NlMsghdr{
			Len:   NLC_MSG_LEN + uint32(dataLen),
			Type:  unix.NLMSG_DONE,
			Flags: 0,
			Seq:   seq,
			Pid:   uint32(os.Getpid()),
		},
		msg: cnMsg{
			id:  cbID{cnIdxProc, cnValProc},
			seq: seq,
			len: uint16(dataLen),
		},
	}
	buf := bytes.NewBuffer(make([]byte, 0, msg.header.Len))
	binary.Write(buf, order, msg)
	binary.Write(buf, order, op)
	return unix.Sendto(nl.fd, buf.Bytes(), 0, sa)
}

func (nl *nlSocket) readEvent() {
	if !nl.binded {
		return
	}

	buf := bufPool.Get().([]byte)
	n, _, err := unix.Recvfrom(nl.fd, buf, 0)
	if err != nil {
		return
	}
	go func() {
		defer bufPool.Put(buf)
		buf = buf[:n]
		for len(buf) > int(NLC_MSG_LEN+PROC_HDR_LEN) {
			hdr := (*unix.NlMsghdr)(unsafe.Pointer(&buf[0]))
			cn := (*cnMsg)(unsafe.Pointer(&buf[unix.NLMSG_HDRLEN]))
			evHdr := (*procEventHeader)(unsafe.Pointer(&buf[NLC_MSG_LEN]))
			switch evHdr.what {
			case PROC_EVENT_NONE:
			case PROC_EVENT_FORK:
				fork := (*forkProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[fork] seq %d, detail %+v\n", cn.seq, fork)
			case PROC_EVENT_EXEC:
				exec := (*execProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[exec] seq %d, detail %+v\n", cn.seq, exec)
			case PROC_EVENT_UID:
				uid := (*idProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[uid] seq %d, detail %+v\n", cn.seq, uid)
			case PROC_EVENT_GID:
				gid := (*idProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[gid] seq %d, detail %+v\n", cn.seq, gid)
			case PROC_EVENT_SID:
				sid := (*sidProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[sid] seq %d, detail %+v\n", cn.seq, sid)
			case PROC_EVENT_PTRACE:
				ptrace := (*ptraceProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[ptrace] seq %d, detail %+v\n", cn.seq, ptrace)
			case PROC_EVENT_COMM:
				comm := (*commProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[comm] seq %d, detail %+v\n", cn.seq, comm)
			case PROC_EVENT_COREDUMP:
				coredump := (*coredumpProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[coredump] seq %d, detail %+v\n", cn.seq, coredump)
			case PROC_EVENT_EXIT:
				exit := (*exitProc)(unsafe.Pointer(&buf[NLC_MSG_LEN+PROC_HDR_LEN]))
				log.Printf("[exit] seq %d, detail %+v\n", cn.seq, exit)
			}
			buf = buf[hdr.Len:]
		}
	}()
}

func getSocket() (*nlSocket, error) {
	sd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_CONNECTOR)
	if err != nil {
		return nil, err
	}
	return &nlSocket{sd, false}, nil
}

func main() {
	nl, err := getSocket()
	if err != nil {
		log.Fatalf("socket: %v\n", err)
	}
	defer nl.Close()

	if err := nl.Connect(); err != nil {
		log.Fatalf("bind: %v\n", err)
	}

	if err := nl.enableMonitor(true); err != nil {
		log.Fatalf("start monitoring: %v\n", err)
	}
	defer nl.enableMonitor(false)
	
	log.Println("monitoring...")
	for {
		nl.readEvent()
	}
}
