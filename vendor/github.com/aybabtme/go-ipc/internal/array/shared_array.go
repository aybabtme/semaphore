// Copyright 2016 Aleksandr Demakin. All rights reserved.

package array

import (
	"math"
	"unsafe"

	"github.com/aybabtme/go-ipc/internal/allocator"
)

const (
	indexEntrySize = unsafe.Sizeof(indexEntry{})
)

type indexEntry struct {
	len     int32
	slotIdx int32
}

type index struct {
	entries []indexEntry
	bitmap  []uint64
	headIdx *int32
}

func bitmapSize(sz int) int {
	bitmapSize := sz / 64
	if sz%64 != 0 {
		bitmapSize++
	}
	return bitmapSize
}

func indexSize(len int) int {
	// len of index, len of bitmap, current head index.
	return len*int(indexEntrySize) + bitmapSize(len)*8 + 4
}

func newIndex(raw unsafe.Pointer, sz int) index {
	bitmapSz := bitmapSize(sz)
	rawIndexSlice := allocator.RawSliceFromUnsafePointer(raw, sz, sz)
	raw = allocator.AdvancePointer(raw, uintptr(sz)*indexEntrySize)
	rawBitmapSlize := allocator.RawSliceFromUnsafePointer(raw, bitmapSz, bitmapSz)
	raw = allocator.AdvancePointer(raw, 8*uintptr(bitmapSz))
	return index{
		entries: *(*[]indexEntry)(rawIndexSlice),
		bitmap:  *(*[]uint64)(rawBitmapSlize),
		headIdx: (*int32)(raw),
	}
}

func lowestZeroBit(value uint64) uint8 {
	for bitIdx := uint8(0); bitIdx < 64; bitIdx++ {
		if value&1 == 0 {
			return bitIdx
		}
		value >>= 1
	}
	return math.MaxUint8
}

func (idx *index) reserveFreeSlot(at int) int32 {
	for i, b := range idx.bitmap {
		if b != math.MaxUint64 {
			bitIdx := lowestZeroBit(b)
			idx.bitmap[i] |= (1 << bitIdx)
			return int32(i*64 + int(bitIdx))
		}
	}
	panic("no free slots")
}

func (idx index) freeSlot(at int) {
	slotIdx := idx.entries[at].slotIdx
	bucketIdx, bitIdx := slotIdx/64, slotIdx%64
	idx.bitmap[bucketIdx] &= ^(1 << uint32(bitIdx))
}

// SharedArray is an array placed in the shared memory with fixed length and element size.
// It is possible to swap elements and pop them from any position. It never moves elements
// in memory, so can be used to implement an array of futexes or spin locks.
type SharedArray struct {
	data *mappedArray
	idx  index
}

// NewSharedArray initializes new shared array with size and element size.
func NewSharedArray(raw unsafe.Pointer, size, elemSize int) *SharedArray {
	data := newMappedArray(raw)
	data.init(size, elemSize)
	idx := newIndex(allocator.AdvancePointer(raw, mappedArrayHdrSize+uintptr(size*elemSize)), size)
	return &SharedArray{
		data: data,
		idx:  idx,
	}
}

// OpenSharedArray opens existing shared array.
func OpenSharedArray(raw unsafe.Pointer) *SharedArray {
	data := newMappedArray(raw)
	idx := newIndex(allocator.AdvancePointer(raw, mappedArrayHdrSize+uintptr(data.cap()*data.elemLen())), data.cap())
	return &SharedArray{
		data: data,
		idx:  idx,
	}
}

// Cap returns array's cpacity
func (arr *SharedArray) Cap() int {
	return arr.data.cap()
}

// Len returns current length.
func (arr *SharedArray) Len() int {
	return arr.data.len()
}

// SafeLen atomically loads returns current length.
func (arr *SharedArray) SafeLen() int {
	return arr.data.safeLen()
}

// ElemSize returns size of the element.
func (arr *SharedArray) ElemSize() int {
	return arr.data.elemLen()
}

// PushBack add new element to the end of the array, merging given datas.
// Returns number of bytes copied, less or equal, than the size of the element.
func (arr *SharedArray) PushBack(datas ...[]byte) int {
	curLen := arr.Len()
	if curLen >= arr.Cap() {
		panic("index out of range")
	}
	physIdx := arr.logicalIdxToPhys(curLen)
	entry := indexEntry{
		slotIdx: arr.idx.reserveFreeSlot(physIdx),
		len:     0,
	}
	slData := arr.data.at(int(entry.slotIdx))
	for _, data := range datas {
		entry.len += int32(copy(slData[entry.len:], data))
		if int(entry.len) < len(data) {
			break
		}
	}
	arr.idx.entries[physIdx] = entry
	arr.data.incLen()
	return int(entry.len)
}

// At returns data at the position i. Returned slice references to the data in the array.
// It does not perform border check.
func (arr *SharedArray) At(i int) []byte {
	entry := arr.entryAt(i)
	return arr.data.at(int(entry.slotIdx))[:int(entry.len)]
}

// AtPointer returns pointer to the data at the position i.
// It does not perform border check.
func (arr *SharedArray) AtPointer(i int) unsafe.Pointer {
	entry := arr.entryAt(i)
	return arr.data.atPointer(int(entry.slotIdx))
}

// PopFront removes the first element of the array.
func (arr *SharedArray) PopFront() {
	curLen := arr.Len()
	if curLen == 0 {
		panic("index out of range")
	}
	arr.idx.freeSlot(int(*arr.idx.headIdx))
	arr.forwardHead()
	arr.data.decLen()
}

// PopBack removes the last element of the array.
func (arr *SharedArray) PopBack() {
	curLen := arr.Len()
	if curLen == 0 {
		panic("index out of range")
	}
	arr.idx.freeSlot(arr.logicalIdxToPhys(curLen - 1))
	if curLen == 1 {
		*arr.idx.headIdx = 0
	}
	arr.data.decLen()
}

// RemoveAt removes i'th element.
func (arr *SharedArray) RemoveAt(i int) {
	curLen := arr.Len()
	if i < 0 || i >= curLen {
		panic("index out of range")
	}
	arr.idx.freeSlot(arr.logicalIdxToPhys(i))
	if i <= curLen/2 {
		for j := i; j > 0; j-- {
			arr.idx.entries[arr.logicalIdxToPhys(j)] = arr.idx.entries[arr.logicalIdxToPhys(j-1)]
		}
		arr.forwardHead()
	} else {
		for j := i; j < curLen-1; j++ {
			arr.idx.entries[arr.logicalIdxToPhys(j)] = arr.idx.entries[arr.logicalIdxToPhys(j+1)]
		}
		if curLen == 1 {
			*arr.idx.headIdx = 0
		}
	}
	arr.data.decLen()
}

// Swap swaps two elements of the array.
func (arr *SharedArray) Swap(i, j int) {
	l := arr.Len()
	if i >= l || j >= l {
		panic("index out of range")
	}
	if i == j {
		return
	}
	i, j = arr.logicalIdxToPhys(i), arr.logicalIdxToPhys(j)
	arr.idx.entries[i], arr.idx.entries[j] = arr.idx.entries[j], arr.idx.entries[i]
}

func (arr *SharedArray) forwardHead() {
	if arr.Len() == 1 {
		*arr.idx.headIdx = 0
	} else {
		*arr.idx.headIdx = (*arr.idx.headIdx + 1) % int32(arr.Cap())
	}
}

func (arr *SharedArray) logicalIdxToPhys(log int) int {
	return (log + int(*arr.idx.headIdx)) % arr.Cap()
}

func (arr *SharedArray) entryAt(log int) indexEntry {
	return arr.idx.entries[arr.logicalIdxToPhys(log)]
}

// CalcSharedArraySize returns the size, needed to place shared array in memory.
func CalcSharedArraySize(size, elemSize int) int {
	return int(mappedArrayHdrSize) + // mq header
		indexSize(size) + // mq index
		size*elemSize // mq messages size
}
