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

	"github.com/believening/netlink-monitor/tools"
	"golang.org/x/sys/unix"
)

var (
	order binary.ByteOrder = tools.Order

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

type connector struct {
	fd     int
	binded bool
}

func (nl *connector) Close() error {
	nl.binded = false
	return unix.Close(nl.fd)
}

func (nl *connector) Connect() error {
	err := unix.Bind(nl.fd, sa)
	nl.binded = (err == nil)
	return err
}

func (nl *connector) enableMonitor(en bool) error {
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

func (nl *connector) readEvent() {
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

func newConnector() (*connector, error) {
	sd, err := tools.NetlinkSocket(syscall.NETLINK_CONNECTOR)
	if err != nil {
		return nil, err
	}
	return &connector{sd, false}, nil
}
