package main

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

// map c struct cb_id defined in connector.h
type cbID struct {
	idx uint32
	val uint32
}

// map c struct cn_msg defined in connector.h
type cnMsg struct {
	id   cbID
	seq  uint32
	ack  uint32
	len  uint32
	data int8
}

func main() {

}
