package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"
)

/*
MotionVectorReader reads the raw motion vectors into a motionVector array
*/
type MotionVectorReader struct {
	Width  int
	Height int
	Mbx    int
	Mby    int
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
NewMotionVectorReader creates a new motion vector reader and begins reading from the reader

math taken from here https://github.com/billw2/pikrellcam/blob/master/src/motion.c#L1634
*/
func NewMotionVectorReader(width int, height int, reader io.ReadCloser) *MotionVectorReader {
	l := new(sync.Mutex)

	ret := &MotionVectorReader{
		Width:  width,
		Height: height,
		Mbx:    1 + (width+15)/16,
		Mby:    1 + height/16,
		reader: reader,
		lock:   l,
		ready:  sync.NewCond(l),
		done:   false,
	}
	go ret.thread()
	return ret
}

func (m *MotionVectorReader) thread() {
	len := m.Mbx * m.Mby
	vect := make([]MotionVector, 2*len)

	i := 0
	for {
		if err := binary.Read(m.reader, binary.LittleEndian, vect[i*len:(i+1)*len]); err != nil {
			log.Print(err)
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
}

/*
Close shuts down the reader and background gofunc
*/
func (m *MotionVectorReader) Close() {
	m.reader.Close()
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
