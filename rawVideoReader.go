package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
)

/*
RawVideoReader is a buffer for the RGB24 video bytes
*/
type RawVideoReader struct {
	Stride int
	Cols   int
	Rows   int

	atom   int32
	frame  []byte
	reader io.ReadCloser
	lock   sync.Locker
	ready  *sync.Cond
	done   bool
}

/*
Done returns true if the reader has been closed
*/
func (rr *RawVideoReader) Done() bool {
	if rr.lock == nil {
		return true
	}

	rr.lock.Lock()
	defer rr.lock.Unlock()

	return rr.done
}

/*
Close stops the thread and closes the wrapped ReadCloser
*/
func (rr *RawVideoReader) Close() error {
	if rr.reader == nil {
		return fmt.Errorf("no reader to close")
	}
	return rr.reader.Close()
}

/*
Start function begins reading from enclosed reader
*/
func (rr *RawVideoReader) Start(reader io.ReadCloser) error {
	oldatom := atomic.SwapInt32(&rr.atom, 1)
	if oldatom == 1 {
		return fmt.Errorf("already started")
	}

	rr.reader = reader
	rr.lock = new(sync.Mutex)
	rr.ready = sync.NewCond(rr.lock)

	bufsize := rr.Stride * rr.Rows
	// double buffer
	buf := make([]byte, 2*bufsize)

	log.Printf("rawvideo bufsize: %d", bufsize)

	var err error

	i := 0
	for {
		_, err = io.ReadFull(rr.reader, buf[i*bufsize:(i+1)*bufsize])

		if err != nil {
			break
		}

		rr.lock.Lock()
		rr.frame = buf[i*bufsize : (i+1)*bufsize]
		rr.lock.Unlock()

		rr.ready.Broadcast()

		i = (i + 1) % 2
	}

	rr.lock.Lock()
	rr.done = true
	rr.lock.Unlock()

	rr.ready.Broadcast()

	// I think we can just reset the atom to put everything back to where it was before the start
	// any wayting threads should end on the condition broadcast with done == true
	atomic.SwapInt32(&rr.atom, 0)
	return err
}

/*
WaitNextFrame will block until the next frame is received from the reader and return that frame and an error if closed
*/
func (rr *RawVideoReader) WaitNextFrame() (*RGB24, error) {
	eof := fmt.Errorf("video completed")
	rr.lock.Lock()
	defer rr.lock.Unlock()

	if rr.done {
		return nil, eof
	}
	rr.ready.Wait()
	if rr.done {
		return nil, eof
	}

	f := make([]byte, len(rr.frame))
	copy(f, rr.frame)

	return FromRaw(f, rr.Stride, rr.Cols, rr.Rows), nil
}

/*
Frame returns the current frame in the buffer
*/
func (rr *RawVideoReader) Frame() *RGB24 {
	rr.lock.Lock()
	defer rr.lock.Unlock()

	if rr.frame == nil {
		return nil
	}

	// log.Printf("length of frame: %d", len(rr.frame))

	f := make([]byte, len(rr.frame))
	copy(f, rr.frame)

	return FromRaw(f, rr.Stride, rr.Cols, rr.Rows)
}
