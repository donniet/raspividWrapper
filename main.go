package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var (
	videoFile  = "video_fifo"
	rawFile    = "raw_fifo"
	motionFile = "motion_fifo"

	rapsividExec = "raspivid"

	width  = 1920
	height = 1080

	defaultBufferSize = 4096

	raspividCommandLine = "\"%s\" -o \"%s\" -x \"%s\" -r \"%s\" -w %d -h %d -rf rgb"
)

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

func (dr *NullReader) Close() {
	dr.r.Close()
}

func NewNullReader(r io.ReadCloser) (ret *NullReader) {
	ret = &NullReader{r}
	go ret.readThread()
	return
}

type VideoReader struct {
	listeners map[io.WriteCloser]bool
	videoPipe io.ReadCloser
	lock      sync.Locker
	bufsize   int
}

func NewVideoReader(r io.ReadCloser) (ret *VideoReader) {
	ret = &VideoReader{
		listeners: make(map[io.WriteCloser]bool),
		videoPipe: r,
		lock:      new(sync.Mutex),
		bufsize:   defaultBufferSize,
	}
	go ret.readThread()
	return
}

func (vr *VideoReader) Close() {
	vr.videoPipe.Close()
}

func (vr *VideoReader) AddListener(w io.WriteCloser) {
	vr.lock.Lock()
	defer vr.lock.Unlock()

	vr.listeners[w] = true
}

func (vr *VideoReader) RemoveListener(w io.WriteCloser) {
	vr.lock.Lock()
	defer vr.lock.Unlock()

	delete(vr.listeners, w)
}

func (vr *VideoReader) readThread() {
	buf := make([]byte, vr.bufsize)

	for {
		n, err := vr.videoPipe.Read(buf)

		if err != nil {
			log.Print(err)
			break
		}

		var toClose []io.WriteCloser
		var toWrite []io.WriteCloser

		vr.lock.Lock()
		for w := range vr.listeners {
			toWrite = append(toWrite, w)
		}
		vr.lock.Unlock()

		for _, w := range toWrite {
			n0 := 0
			for n0 < n {
				n1, err := w.Write(buf[n0:n])
				if err != nil {
					toClose = append(toClose, w)
					break
				}
				n0 += n1
			}
		}

		vr.lock.Lock()
		for _, w := range toClose {
			w.Close() // ignore error
			delete(vr.listeners, w)
		}
		vr.lock.Unlock()
	}

	vr.lock.Lock()
	for w := range vr.listeners {
		w.Close()
		delete(vr.listeners, w)
	}
	vr.lock.Unlock()
}

func main() {
	log.Printf("looking up raspivid path")
	raspividPath, err := exec.LookPath(rapsividExec)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("creating named pipes")
	for _, f := range []string{videoFile, rawFile, motionFile} {
		err := syscall.Mkfifo(f, 0660)
		if err != nil {
			log.Fatal(err)
		}

		defer os.Remove(f)
	}

	log.Printf("handling interrupt")
	interrupted := make(chan os.Signal)
	signal.Notify(interrupted, os.Interrupt)

	log.Printf("starting raspivid")
	cmd := exec.Command(raspividPath, "-o", videoFile, "-r", rawFile, "-x", motionFile, "-w", fmt.Sprintf("%d", width), "-h", fmt.Sprintf("%d", height), "-rf", "rgb")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("opening named pipes for reading")
	videoPipe, err := os.OpenFile(videoFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open video file '%s': %v", videoFile, err)
	}
	rawPipe, err := os.OpenFile(rawFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open raw file '%s': %v", rawFile, err)
	}
	motionPipe, err := os.OpenFile(motionFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open motion file '%s': %v", motionFile, err)
	}

	log.Printf("starting readers")
	motionReader := NewNullReader(motionPipe)
	rawReader := NewNullReader(rawPipe)
	videoReader := NewVideoReader(videoPipe)

	defer videoReader.Close()
	defer motionReader.Close()
	defer rawReader.Close()

	<-interrupted
	fmt.Println("Closing")

	cmd.Process.Signal(os.Interrupt)
	cmd.Wait()
}
