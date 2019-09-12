package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var (
	rawFile    = "raw_fifo"
	motionFile = "motion_fifo"

	raspividExec = "raspivid"

	videoPort = ":3000"
	restAddr  = ":8888"

	width        = 1640
	height       = 1232
	bitrate      = 17000000
	framerate    = 25
	keyFrameRate = 12
	analogGain   = 4.0
	digitalGain  = 1.0
	refreshType  = "both"
	h264Level    = "4.2"
	h264Profile  = "main"

	defaultBufferSize = 2048
)

func init() {
	flag.StringVar(&rawFile, "raw", rawFile, "raw video fifo path to be created - do not name this an existing file name!")
	flag.StringVar(&motionFile, "motion", motionFile, "motion vectors fifo path to be created - do not name this an existing file name!")
	flag.StringVar(&raspividExec, "raspivid", raspividExec, "name of raspivid executable to be located in your PATH")
	flag.StringVar(&videoPort, "vidaddr", videoPort, "address to listen for video connections")
	flag.IntVar(&width, "width", width, "width of video")
	flag.IntVar(&width, "w", width, "width of video")
	flag.IntVar(&height, "height", height, "height of video")
	flag.IntVar(&height, "h", height, "height of video")
	flag.IntVar(&bitrate, "bitrate", bitrate, "bitrate of h264 video")
	flag.IntVar(&framerate, "fps", framerate, "framerate requested from raspivid")
	flag.IntVar(&keyFrameRate, "keyfps", keyFrameRate, "key frame rate of h264 video")
	flag.Float64Var(&analogGain, "ag", analogGain, "analog gain sent to raspivid")
	flag.Float64Var(&digitalGain, "dg", digitalGain, "digital gain sent to raspivid")
	flag.StringVar(&refreshType, "refreshType", refreshType, "intra refresh type (cyclic, adaptive, both, cyclicrows)")
	flag.StringVar(&h264Level, "h264level", h264Level, "h264 encoder level (4, 4.1, 4.2)")
	flag.StringVar(&h264Profile, "h264profile", h264Profile, "h264 encoder profile (baseline, main, high)")
	flag.StringVar(&restAddr, "restaddr", restAddr, "address of rest interface")

}

type RawVideoReader struct {
	width    int
	height   int
	channels int

	frame  []byte
	reader io.ReadCloser
	lock   sync.Locker
}

func NewRawVideoReader(width int, height int, channels int, reader io.ReadCloser) *RawVideoReader {
	ret := &RawVideoReader{
		width:    width,
		height:   height,
		channels: channels,
		reader:   reader,
		lock:     new(sync.Mutex),
	}
	go ret.readThread()
	return ret
}

func (rr *RawVideoReader) Close() {
	rr.reader.Close()
}

func (rr *RawVideoReader) readThread() {
	buf := make([]byte, rr.width*rr.height*rr.channels)
	for {
		_, err := io.ReadFull(rr.reader, buf)

		if err != nil {
			break
		}

		rr.lock.Lock()
		rr.frame = buf
		rr.lock.Unlock()
	}
}

func (rr *RawVideoReader) Frame() image.Image {
	rr.lock.Lock()
	defer rr.lock.Unlock()

	if rr.frame == nil {
		return nil
	}

	return FromRaw(rr.frame, rr.width*rr.channels)
}

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

type socketServer struct {
	connections map[*net.TCPConn]bool
	lock        sync.Locker
	listener    *net.TCPListener
}

func newSocketServer() *socketServer {
	return &socketServer{
		connections: make(map[*net.TCPConn]bool),
		lock:        new(sync.Mutex),
	}
}

func (s *socketServer) serve(port string) error {
	addr, err := net.ResolveTCPAddr("tcp", port)
	if err != nil {
		log.Fatal("addr invalid tcp: '%s': %v", addr, err)
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = ln

	go func() {
		defer ln.Close()

		for {
			conn, err := s.listener.AcceptTCP()
			if err != nil {
				return
			}
			if err := conn.SetKeepAlive(true); err != nil {
				log.Printf("error setting keepalive: %v", err)
				// continue
			}
			log.Printf("accepted connection %s", conn.RemoteAddr())

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

	var conns []*net.TCPConn
	s.lock.Lock()
	for c := range s.connections {
		conns = append(conns, c)
	}
	s.lock.Unlock()

	toRemove := make(map[*net.TCPConn]bool)
	for _, c := range conns {
		n := 0
		for n < len(b) {
			// log.Printf("writing to %s", c.RemoteAddr())
			n0, err := c.Write(b[n:])
			if err != nil || n0 == 0 {
				log.Printf("removing connection %s err: %v", c.RemoteAddr(), err)
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
	flag.Parse()

	log.Printf("looking up raspivid path")
	raspividPath, err := exec.LookPath(raspividExec)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("creating named pipes")
	for _, f := range []string{rawFile, motionFile} {
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
	cmd := exec.Command(raspividPath,
		"-t", "0",
		"-b", fmt.Sprintf("%d", bitrate),
		"-o", "-",
		"-fps", fmt.Sprintf("%d", framerate),
		"-rf", "rgb",
		"-r", rawFile,
		"-x", motionFile,
		"-w", fmt.Sprintf("%d", width),
		"-h", fmt.Sprintf("%d", height),
		"-stm",
		"-ih",
		"-ag", fmt.Sprintf("%f", analogGain),
		"-dg", fmt.Sprintf("%f", digitalGain),
		"-g", fmt.Sprintf("%d", keyFrameRate),
		"-if", refreshType,
		"-ih",
		"-lev", h264Level,
		"-pf", h264Profile,
	)

	videoPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("error getting stdout: %v", err)
	}

	videoErrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("error getting stderr: %v", err)
	}
	go func() {
		r := bufio.NewReader(videoErrPipe)
		for {
			ln, err := r.ReadBytes('\n')
			if err != nil {
				break
			}
			log.Printf("raspivid: %s", ln)
		}
	}()

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

	var rawPipe, motionPipe *os.File

	motionPipe, err = os.OpenFile(motionFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open motion file '%s': %v", motionFile, err)
	}
	rawPipe, err = os.OpenFile(rawFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open raw file '%s': %v", rawFile, err)
	}

	log.Printf("starting readers")
	motionReader := NewNullReader(motionPipe)
	rawReader := NewRawVideoReader(width, height, 3, rawPipe)

	sock := newSocketServer()
	sock.serve(videoPort)

	go func() {
		buf := make([]byte, defaultBufferSize)
		for {
			n, err := videoPipe.Read(buf)
			if err != nil {
				break
			}
			sock.Write(buf[:n])
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/frame.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/jpeg")
		if err := jpeg.Encode(w, rawReader.Frame(), nil); err != nil {
			log.Printf("error encoding frame: %v", err)
		}
	})
	server := &http.Server{
		Addr:    restAddr,
		Handler: mux,
	}
	defer server.Shutdown(context.Background())
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
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
