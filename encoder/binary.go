package encoder

import "math"

func _toByteSlice(number uint64, bytes int) []byte {
	var b []byte
	for i := bytes - 1; i >= 0; i-- {
		shifted := uint8((number >> (i * 8)) & 0xff)
		b = append(b, shifted)
	}
	return b
}

func u16to8(number uint16) []byte {
	return _toByteSlice(uint64(number), 2)
}

func u32to8(number uint32) []byte {
	return _toByteSlice(uint64(number), 4)
}

func u64to8(number uint64) []byte {
	return _toByteSlice(number, 8)
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

func _fromByteSlice(slice []byte) uint64 {
	total := uint64(0)
	l := len(slice) - 1
	for i := l; i >= 0; i-- {
		total += uint64(float64(slice[l-i]) * math.Pow(16, float64(i*2)))
	}

	return total
}

func SliceToU16(slice []byte) uint16 {
	return uint16(_fromByteSlice(slice))
}

func SliceToU32(slice []byte) uint32 {
	return uint32(_fromByteSlice(slice))
}

func SliceToU64(slice []byte) uint64 {
	return _fromByteSlice(slice)
}

func addLengthMarker(length int, extended bool, slice *[]byte) *[]byte {
	if extended {
		return Add64ToSlice(uint64(length), slice)
	} else {
		return Add32ToSlice(uint32(length), slice)
	}
}
