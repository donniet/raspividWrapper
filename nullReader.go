package main

import (
	"fmt"
	"io"
	"sync/atomic"
)

/*
NullReader reads and throws away the data from the wrapped reader
*/
type NullReader struct {
	atom int32
	r    io.ReadCloser
}

/*
Start begin reading (and doing nothing with) the data from reader
*/
func (dr *NullReader) Start(reader io.ReadCloser) (err error) {
	old := atomic.SwapInt32(&dr.atom, 1)
	if old == 1 {
		return fmt.Errorf("Start already called")
	}

	dr.r = reader
	defer reader.Close()

	buf := make([]byte, defaultBufferSize)

	for {
		_, err = dr.r.Read(buf)

		if err != nil {
			break
		}
	}

	atomic.SwapInt32(&dr.atom, 0)
	return
}

/*
Close closes the wrapped reader
*/
func (dr *NullReader) Close() error {
	if dr.r == nil {
		return fmt.Errorf("not started")
	}
	return dr.r.Close()
}
