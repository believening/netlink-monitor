package main

import "golang.org/x/sys/unix"

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
