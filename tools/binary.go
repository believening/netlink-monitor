package tools

import (
	"encoding/binary"
	"unsafe"
)

var Order binary.ByteOrder

func init() {
	x := 0x1001
	if *(*byte)(unsafe.Pointer(&x)) == 0x10 {
		Order = binary.BigEndian
	} else {
		Order = binary.LittleEndian
	}
}
