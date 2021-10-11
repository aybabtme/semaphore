// Copyright 2015 Aleksandr Demakin. All rights reserved.

package shm

import (
	"os"
	"runtime"

	"github.com/aybabtme/go-ipc/internal/common"
	"github.com/aybabtme/go-ipc/mmf"
	"github.com/pkg/errors"
)

var (
	_ SharedMemoryObject = (*MemoryObject)(nil)
)

// SharedMemoryObject is an interface, which must be implemented
// by any implemetation of an object used for mapping into memory.
type SharedMemoryObject interface {
	Size() int64
	Truncate(size int64) error
	Close() error
	Destroy() error
	mmf.Mappable
}

// MemoryObject represents an object which can be used to
// map shared memory regions into the process' address space.
type MemoryObject struct {
	*memoryObject
}

// NewMemoryObject creates a new shared memory object.
//	name - a name of the object. should not contain '/' and exceed 255 symbols (30 on darwin).
//	size - object size.
//	flag - flag is a combination of open flags from 'os' package.
//	perm - object's permission bits.
func NewMemoryObject(name string, flag int, perm os.FileMode) (*MemoryObject, error) {
	impl, err := newMemoryObject(name, flag, perm)
	if err != nil {
		return nil, err
	}
	result := &MemoryObject{impl}
	runtime.SetFinalizer(impl, func(memObject *memoryObject) {
		memObject.Close()
	})
	return result, nil
}

// NewMemoryObjectSize opens or creates a shared memory object with the given name.
// If the object was created, it is truncated to 'size'.
// Otherwise, checks, that the existing object is at least 'size' bytes long.
// Returns an object, true, if it was created, and an error.
func NewMemoryObjectSize(name string, flag int, perm os.FileMode, size int64) (SharedMemoryObject, bool, error) {
	var obj *MemoryObject
	creator := func(create bool) error {
		var err error
		creatorFlag := os.O_RDWR
		if create {
			creatorFlag |= (os.O_CREATE | os.O_EXCL)
		}
		obj, err = NewMemoryObject(name, creatorFlag, perm)
		return errors.Cause(err)
	}
	created, resultErr := common.OpenOrCreate(creator, flag)
	if resultErr != nil {
		return nil, false, resultErr
	}
	if created {
		if resultErr = obj.Truncate(size); resultErr != nil {
			return nil, false, resultErr
		}
	} else if obj.Size() < size {
		return nil, false, errors.Errorf("existing object is smaller (%d), than needed(%d)", obj.Size(), size)
	}
	return obj, created, nil
}

// Destroy closes the object and removes it permanently.
func (obj *MemoryObject) Destroy() error {
	return obj.memoryObject.Destroy()
}

// Name returns the name of the object as it was given to NewMemoryObject().
func (obj *MemoryObject) Name() string {
	return obj.memoryObject.Name()
}

// Close closes object's underlying file object.
// Darwin: a call to Close() causes invalid argument error,
// if the object was not truncated. So, in this case we return nil as an error.
func (obj *MemoryObject) Close() error {
	return obj.memoryObject.Close()
}

// Truncate resizes the shared memory object.
// Darwin: it is possible to truncate an object only once after it was created.
// Darwin: the size should be divisible by system page size,
// otherwise the size is set to the nearest page size divider greater, then the given size.
func (obj *MemoryObject) Truncate(size int64) error {
	return obj.memoryObject.Truncate(size)
}

// Size returns the current object size.
// Darwin: it may differ from the size passed passed to Truncate.
func (obj *MemoryObject) Size() int64 {
	return obj.memoryObject.Size()
}

// Fd returns a descriptor of the object's underlying file object.
func (obj *MemoryObject) Fd() uintptr {
	return obj.memoryObject.Fd()
}

// DestroyMemoryObject permanently removes given memory object.
func DestroyMemoryObject(name string) error {
	return destroyMemoryObject(name)
}
