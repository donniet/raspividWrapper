package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

/*
MotionVectorReader reads the raw motion vectors into a motionVector array
*/
type MotionVectorReader struct {
	MBX int
	MBY int

	atom   int32
	reader io.ReadCloser
	buffer []MotionVector
	lock   sync.Locker
	ready  *sync.Cond
	done   bool
}

/*
MotionVector represents a single motion vector on a superblock, and is a binary compatible format
*/
type MotionVector struct {
	X   int8
	Y   int8
	Sad int16
}

/*
Start begins reading motion vectors from the readers
*/
func (m *MotionVectorReader) Start(reader io.ReadCloser) error {
	oldatom := atomic.SwapInt32(&m.atom, 1)
	if oldatom == 1 {
		return fmt.Errorf("start already running")
	}

	m.reader = reader
	defer reader.Close()
	m.lock = new(sync.Mutex)
	m.ready = sync.NewCond(m.lock)

	len := m.MBX * m.MBY
	vect := make([]MotionVector, 2*len)

	i := 0

	var err error

	for {
		if err = binary.Read(m.reader, binary.LittleEndian, vect[i*len:(i+1)*len]); err != nil {
			break
		}

		m.lock.Lock()
		m.buffer = vect[i*len : (i+1)*len]
		m.lock.Unlock()

		m.ready.Broadcast()

		i = (i + 1) % 2
	}

	m.lock.Lock()
	m.done = true
	m.lock.Unlock()

	m.ready.Broadcast()

	// again I think this atomic thingy should work to reset everything and allow start to be called again
	// should probably be tested...
	atomic.SwapInt32(&m.atom, 0)

	return err
}

/*
Close shuts down the reader and background gofunc
*/
func (m *MotionVectorReader) Close() error {
	if m.reader == nil {
		return fmt.Errorf("not started")
	}
	return m.reader.Close()
}

/*
WaitNextMotionVectors waits for the next set of motion vectors then returns them or an error if there are no more
*/
func (m *MotionVectorReader) WaitNextMotionVectors() ([]MotionVector, error) {
	eof := fmt.Errorf("completed thread")
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.done {
		return nil, eof
	}
	m.ready.Wait()
	if m.done {
		return nil, eof
	}
	ret := make([]MotionVector, len(m.buffer))
	copy(ret, m.buffer)
	return ret, nil
}

/*
Done returns true of the reader has been shutdown
*/
func (m *MotionVectorReader) Done() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.done
}

/*
MotionVectors gets the most recent motion vectors from the buffer
*/
func (m *MotionVectorReader) MotionVectors() []MotionVector {
	m.lock.Lock()
	defer m.lock.Unlock()

	ret := make([]MotionVector, len(m.buffer))
	copy(ret, m.buffer)

	return ret
}
