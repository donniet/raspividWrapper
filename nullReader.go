package main

import (
	"io"
)

/*
NullReader reads and throws away the data from the wrapped reader
*/
type NullReader struct {
	r io.ReadCloser
}

func (dr *NullReader) readThread() {
	buf := make([]byte, defaultBufferSize)

	for {
		_, err := dr.r.Read(buf)

		if err != nil {
			break
		}
	}
}

/*
Close closes the wrapped reader
*/
func (dr *NullReader) Close() {
	dr.r.Close()
}

/*
NewNullReader creates a NullReader which wrapps the passed ReadCloser
*/
func NewNullReader(r io.ReadCloser) (ret *NullReader) {
	ret = &NullReader{r}
	go ret.readThread()
	return
}