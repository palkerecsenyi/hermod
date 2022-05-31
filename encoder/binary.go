package encoder

import (
	"encoding/binary"
)

func u16to8(number uint16) []byte {
	slice := make([]byte, 2)
	binary.BigEndian.PutUint16(slice, number)
	return slice
}

func u32to8(number uint32) []byte {
	slice := make([]byte, 4)
	binary.BigEndian.PutUint32(slice, number)
	return slice
}

func u64to8(number uint64) []byte {
	slice := make([]byte, 8)
	binary.BigEndian.PutUint64(slice, number)
	return slice
}

func Add64ToSlice(number uint64, slice *[]byte) *[]byte {
	newSlice := append(*slice, u64to8(number)...)
	return &newSlice
}

func Add32ToSlice(number uint32, slice *[]byte) *[]byte {
	newSlice := append(*slice, u32to8(number)...)
	return &newSlice
}

func Add16ToSlice(number uint16, slice *[]byte) *[]byte {
	newSlice := append(*slice, u16to8(number)...)
	return &newSlice
}

func SliceToU16(slice []byte) uint16 {
	return binary.BigEndian.Uint16(slice)
}

func SliceToU32(slice []byte) uint32 {
	return binary.BigEndian.Uint32(slice)
}

func SliceToU64(slice []byte) uint64 {
	return binary.BigEndian.Uint64(slice)
}

func addLengthMarker(length int, extended bool, slice *[]byte) *[]byte {
	if extended {
		return Add64ToSlice(uint64(length), slice)
	} else {
		return Add32ToSlice(uint32(length), slice)
	}
}
