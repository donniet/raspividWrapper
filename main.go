package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

type socketServer struct {
	connections map[net.Conn]bool
	lock        sync.Locker
	listener    net.Listener
}

func newSocketServer() *socketServer {
	return &socketServer{
		connections: make(map[net.Conn]bool),
		lock:        new(sync.Mutex),
	}
}

func (s *socketServer) serve(port string) error {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	s.listener = ln

	go func() {
		defer ln.Close()

		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}

			s.lock.Lock()
			s.connections[conn] = true
			s.lock.Unlock()
		}
	}()

	return nil
}

// this multiplexing is happening in two places-- in the reader and here.  I don't know which is the best place...
func (s *socketServer) Write(b []byte) (int, error) {
	// maybe make this asynchronous?

	var conns []net.Conn
	s.lock.Lock()
	for c := range s.connections {
		conns = append(conns, c)
	}
	s.lock.Unlock()

	toRemove := make(map[net.Conn]bool)
	for _, c := range conns {
		n := 0
		for n < len(b) {
			n0, err := c.Write(b)
			if err != nil {
				toRemove[c] = true
				c.Close()
				break
			}
			n += n0
		}
	}

	s.lock.Lock()
	for c := range toRemove {
		delete(s.connections, c)
	}
	s.lock.Unlock()
	return len(b), nil
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
	cmd := exec.Command(raspividPath, "-t", "0", "-o", videoFile, "-r", rawFile, "-x", motionFile, "-w", fmt.Sprintf("%d", width), "-h", fmt.Sprintf("%d", height), "-rf", "rgb")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	processEnded := make(chan bool)

	go func() {
		cmd.Wait()
		processEnded <- true
	}()

	log.Printf("opening named pipes for reading")

	// we don't know the order that raspivid will open the files, and so we'll open these in gofuncs

	var videoPipe, rawPipe, motionPipe *os.File

	counter := make(chan bool)

	go func() {
		var err error
		log.Printf("trying to open video")
		videoPipe, err = os.OpenFile(videoFile, os.O_CREATE, os.ModeNamedPipe)
		if err != nil {
			log.Fatalf("could not open video file '%s': %v", videoFile, err)
		}
		counter <- true
		log.Printf("video opened")
	}()
	go func() {
		log.Printf("trying to open raw")
		var err error
		rawPipe, err = os.OpenFile(rawFile, os.O_CREATE, os.ModeNamedPipe)
		if err != nil {
			log.Fatalf("could not open raw file '%s': %v", rawFile, err)
		}
		counter <- true
		log.Printf("opened raw.")
	}()
	go func() {
		log.Printf("trying to open motion")
		var err error
		motionPipe, err = os.OpenFile(motionFile, os.O_CREATE, os.ModeNamedPipe)
		if err != nil {
			log.Fatalf("could not open motion file '%s': %v", motionFile, err)
		}
		counter <- true
		log.Printf("opened motion")
	}()

	timeout := time.NewTimer(10000 * time.Millisecond)
	for c := 0; c < 3; {
		select {
		case <-counter:
			c++
		case <-timeout.C:
			log.Fatal("did not open the video, motion and raw files fast enough")
		}
	}

	log.Printf("starting readers")
	motionReader := NewNullReader(motionPipe)
	rawReader := NewNullReader(rawPipe)

	sock := newSocketServer()
	sock.serve(":3000")

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := videoPipe.Read(buf)
			if err != nil {
				break
			}
			sock.Write(buf[:n])
		}
	}()

	// videoReader := NewVideoReader(videoPipe)

	// defer videoReader.Close()
	defer motionReader.Close()
	defer rawReader.Close()

	killProcess := true
	select {
	case <-interrupted:
		break
	case <-processEnded:
		killProcess = false
		break
	}

	fmt.Println("Closing")

	if killProcess {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}
}
