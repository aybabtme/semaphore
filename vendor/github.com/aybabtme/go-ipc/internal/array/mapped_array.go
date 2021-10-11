// Copyright 2016 Aleksandr Demakin. All rights reserved.

package array

import (
	"sync/atomic"
	"unsafe"

	"github.com/aybabtme/go-ipc/internal/allocator"
)

const (
	mappedArrayHdrSize = unsafe.Sizeof(mappedArray{})
)

type mappedArray struct {
	capacity       int32
	elemSize       int32
	size           int32
	dummyDataArray [0]byte
}

func newMappedArray(pointer unsafe.Pointer) *mappedArray {
	return (*mappedArray)(pointer)
}

func (arr *mappedArray) init(capacity, elemSize int) {
	arr.capacity = int32(capacity)
	arr.elemSize = int32(elemSize)
	arr.size = 0
}

func (arr *mappedArray) elemLen() int {
	return int(arr.elemSize)
}

func (arr *mappedArray) cap() int {
	return int(arr.capacity)
}

func (arr *mappedArray) len() int {
	return int(arr.size)
}

func (arr *mappedArray) safeLen() int {
	return int(atomic.LoadInt32(&arr.size))
}

func (arr *mappedArray) incLen() {
	atomic.AddInt32(&arr.size, 1)
}

func (arr *mappedArray) decLen() {
	atomic.AddInt32(&arr.size, -1)
}

func (arr *mappedArray) atPointer(idx int) unsafe.Pointer {
	return allocator.AdvancePointer(unsafe.Pointer(&arr.dummyDataArray), uintptr(idx*int(arr.elemSize)))
}

func (arr *mappedArray) at(idx int) []byte {
	slotsPtr := arr.atPointer(idx)
	return allocator.ByteSliceFromUnsafePointer(slotsPtr, int(arr.elemSize), int(arr.elemSize))
}
