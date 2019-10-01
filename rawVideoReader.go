package main

import (
	"fmt"
	"io"
	"log"
	"sync"
)

/*
RawVideoReader is a buffer for the RGB24 video bytes
*/
type RawVideoReader struct {
	stride int
	cols   int
	rows   int

	frame  []byte
	reader io.ReadCloser
	lock   sync.Locker
	ready  *sync.Cond
	done   bool
}

/*
NewRawVideoReader creates a new reader from the stride, cols and rows and a ReadCloser and starts reading data in a gofunc
*/
func NewRawVideoReader(stride int, cols int, rows int, reader io.ReadCloser) *RawVideoReader {
	l := new(sync.Mutex)
	ret := &RawVideoReader{
		stride: stride,
		cols:   cols,
		rows:   rows,
		reader: reader,
		lock:   l,
		ready:  sync.NewCond(l),
		done:   false,
	}
	go ret.readThread()
	return ret
}

/*
Done returns true if the reader has been closed
*/
func (rr *RawVideoReader) Done() bool {
	rr.lock.Lock()
	defer rr.lock.Unlock()

	return rr.done
}

/*
Close stops the thread and closes the wrapped ReadCloser
*/
func (rr *RawVideoReader) Close() {
	rr.reader.Close()
}

func (rr *RawVideoReader) readThread() {
	bufsize := rr.stride * rr.rows
	// double buffer
	buf := make([]byte, 2*bufsize)

	log.Printf("rawvideo bufsize: %d", bufsize)

	i := 0
	for {
		_, err := io.ReadFull(rr.reader, buf[i*bufsize:(i+1)*bufsize])

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

	return FromRaw(f, rr.stride, rr.cols, rr.rows), nil
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

	log.Printf("length of frame: %d", len(rr.frame))

	f := make([]byte, len(rr.frame))
	copy(f, rr.frame)

	return FromRaw(f, rr.stride, rr.cols, rr.rows)
}
