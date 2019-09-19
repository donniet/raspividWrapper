package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/donniet/raspividWrapper/videoService"
	empty "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var (
	rawFile    = "raw_fifo"
	motionFile = "motion_fifo"

	raspividExec = "raspivid"

	videoPort = ":3000"
	restAddr  = ":8888"
	grpcAddr  = ":5555"

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
	jpegQuality  = 85

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
	flag.StringVar(&grpcAddr, "grpcaddr", grpcAddr, "address of GRPC interface")

}

type VideoServerGRPC struct {
	MotionReader *MotionVectorReader
	VideoReader  *RawVideoReader
}

func (v *VideoServerGRPC) MotionRaw(_ *empty.Empty, rawServer videoService.Video_MotionRawServer) error {
	for {
		vects, err := v.MotionReader.WaitNextMotionVectors()

		if err != nil {
			break
		}

		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.LittleEndian, vects); err != nil {
			log.Fatalf("error writing vectors to binary: %v", err)
		}

		rawServer.Send(&videoService.Frame{Data: buf.Bytes()})
	}

	return nil
}

func (v *VideoServerGRPC) FrameJPEG(ctx context.Context, _ *empty.Empty) (*videoService.Frame, error) {
	frame := v.VideoReader.Frame()

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, frame, &jpeg.Options{Quality: jpegQuality}); err != nil {
		log.Fatalf("error encoding JPEG to buffer: %v", err)
	}

	return &videoService.Frame{Data: buf.Bytes()}, nil
}

func (v *VideoServerGRPC) VideoJPEG(_ *empty.Empty, rawServer videoService.Video_VideoJPEGServer) (err error) {
	for {
		frame, err2 := v.VideoReader.WaitNextFrame()

		if err2 != nil {
			err = err2
			break
		}

		// should probably use a cached version of the jpeg
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, frame, &jpeg.Options{Quality: jpegQuality}); err != nil {
			// probably shouldn't be fatal
			log.Fatalf("error encoding JPEG to buffer: %v", err)
		}

		rawServer.Send(&videoService.Frame{Data: buf.Bytes()})
	}

	return nil
}

func (v *VideoServerGRPC) FrameRaw(ctx context.Context, _ *empty.Empty) (*videoService.Frame, error) {
	frame := v.VideoReader.Frame()

	rgb := frame.(*RGB24)

	return &videoService.Frame{Data: rgb.Pix}, nil
}

func (v *VideoServerGRPC) VideoRaw(_ *empty.Empty, rawServer videoService.Video_VideoRawServer) error {
	for {
		frame, err := v.VideoReader.WaitNextFrame()

		if err != nil {
			break
		}

		rgb, ok := frame.(*RGB24)
		if !ok {
			log.Fatal("image is not RGB24")
		}

		rawServer.Send(&videoService.Frame{Data: rgb.Pix})
	}

	return nil
}

func (v *VideoServerGRPC) MetaData(ctx context.Context, _ *empty.Empty) (*videoService.VideoMetaData, error) {
	return &videoService.VideoMetaData{
		Size: &videoService.Rectangle{
			X: int32(width),
			Y: int32(height),
		},
		MacroBlocks: &videoService.Rectangle{
			X: int32(1 + (width+15)/16),
			Y: int32(1 + height/16),
		},
		BitRate:      int32(bitrate),
		FrameRate:    int32(framerate),
		KeyFrameRate: int32(keyFrameRate),
		AnalogGain:   float32(analogGain),
		DigitalGain:  float32(digitalGain),
		RefreshType:  refreshType,
		H264Level:    h264Level,
		H264Profile:  h264Profile,
	}, nil
}

type MotionVectorReader struct {
	Width  int
	Height int
	Mbx    int
	Mby    int
	reader io.ReadCloser
	buffer []motionVector
	lock   sync.Locker
	ready  *sync.Cond
	done   bool
}

type motionVector struct {
	X   int8
	Y   int8
	Sad int16
}

// math taken from here https://github.com/billw2/pikrellcam/blob/master/src/motion.c#L1634
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
	vect := make([]motionVector, 2*len)

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

func (m *MotionVectorReader) Close() {
	m.reader.Close()
}

/*
WaitNextMotionVectors waits for the next set of motion vectors then returns them or an error if there are no more
*/
func (m *MotionVectorReader) WaitNextMotionVectors() ([]motionVector, error) {
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
	ret := make([]motionVector, len(m.buffer))
	copy(ret, m.buffer)
	return ret, nil
}

func (m *MotionVectorReader) Done() bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.done
}

func (m *MotionVectorReader) MotionVectors() []motionVector {
	m.lock.Lock()
	defer m.lock.Unlock()

	ret := make([]motionVector, len(m.buffer))
	copy(ret, m.buffer)

	return ret
}

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

func (rr *RawVideoReader) Done() bool {
	rr.lock.Lock()
	defer rr.lock.Unlock()

	return rr.done
}

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

func (rr *RawVideoReader) WaitNextFrame() (image.Image, error) {
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

func (rr *RawVideoReader) Frame() image.Image {
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
		"-n",
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
		close(processEnded)
	}()
	defer func() {
		if !cmd.ProcessState.Exited() {
			cmd.Process.Signal(os.Interrupt)
			cmd.Wait()
		}
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
	motionReader := NewMotionVectorReader(width, height, motionPipe)
	stride := 3 * width
	if r := stride % 16; r != 0 {
		stride += 3 * r
	}
	rawReader := NewRawVideoReader(stride, width, height, rawPipe)

	defer motionReader.Close()
	defer rawReader.Close()

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
	serveJPEG := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/jpeg")
		frame := rawReader.Frame()
		log.Printf("frame size: %v", frame.Bounds())
		if err := jpeg.Encode(w, frame, nil); err != nil {
			log.Printf("error encoding frame: %v", err)
		}
	})
	mux.Handle("/frame.jpg", serveJPEG)
	mux.Handle("/frame.jpeg", serveJPEG)
	mux.HandleFunc("/frame.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/png")
		if err := png.Encode(w, rawReader.Frame()); err != nil {
			log.Printf("error encoding frame: %v", err)
		}
	})
	mux.HandleFunc("/frame.rgb", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/octet-stream")
		frame := rawReader.Frame()
		rgb := frame.(*RGB24)
		w.Header().Add("X-Image-Stride", fmt.Sprintf("%d", rgb.Stride))
		w.Header().Add("X-Image-Rows", fmt.Sprintf("%d", rgb.Rect.Dy()))
		w.Header().Add("X-Image-Cols", fmt.Sprintf("%d", rgb.Rect.Dx()))

		n, err := io.Copy(w, bytes.NewReader(rgb.Pix))
		if err != nil {
			log.Printf("error writing raw image: %v", err)
		} else if n < int64(len(rgb.Pix)) {
			log.Printf("only %d bytes written out of %d", n, len(rgb.Pix))
		}
	})
	mux.HandleFunc("/motion.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/octet-stream")
		vectors := motionReader.MotionVectors()
		binary.Write(w, binary.LittleEndian, vectors)
	})
	mux.HandleFunc("/config.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(map[string]interface{}{
			"videoAddr":    videoPort,
			"restAddr":     restAddr,
			"width":        width,
			"height":       height,
			"bitrate":      bitrate,
			"framerate":    framerate,
			"keyFrameRate": keyFrameRate,
			"analogGain":   analogGain,
			"digitalGain":  digitalGain,
			"refreshType":  refreshType,
			"h264Level":    h264Level,
			"h264Profile":  h264Profile,
			"mbx":          motionReader.Mbx,
			"mby":          motionReader.Mby,
		})
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

	grpcServer := &VideoServerGRPC{
		MotionReader: motionReader,
		VideoReader:  rawReader,
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	videoService.RegisterVideoServer(s, grpcServer)
	go func() {
		log.Printf("starting GRPC server on '%s'", grpcAddr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	defer s.GracefulStop()

	select {
	case <-interrupted:
		break
	case <-processEnded:
		break
	}

	fmt.Println("Closing")
}
