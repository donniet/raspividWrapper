package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/donniet/raspividWrapper/videoService"
	"google.golang.org/grpc"
)

var (
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

	tempDir = "/tmp"
)

func init() {
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
	flag.StringVar(&tempDir, "tempdir", tempDir, "temporary directory root")

}

func main() {
	flag.Parse()

	log.Printf("looking up raspivid path")
	raspividPath, err := exec.LookPath(raspividExec)
	if err != nil {
		log.Fatal(err)
	}

	fifoDirectory, err := ioutil.TempDir(tempDir, "raspividWrapper_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(fifoDirectory)

	rawPath := filepath.Join(fifoDirectory, "raw")
	motionPath := filepath.Join(fifoDirectory, "motion")

	log.Printf("creating named pipes")
	for _, f := range []string{rawPath, motionPath} {
		// wrap in anon func for scope for the defer
		err := syscall.Mkfifo(f, 0660)
		if err != nil {
			log.Printf("WARNING: '%s': %v", f, err)
		}
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
		"-r", rawPath,
		"-x", motionPath,
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

	motionPipe, err = os.OpenFile(motionPath, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open motion file '%s': %v", motionPath, err)
	}
	rawPipe, err = os.OpenFile(rawPath, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("could not open raw file '%s': %v", rawPath, err)
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
		w.Header().Add("Access-Control-Allow-Origin", "*")
		frame := rawReader.Frame()
		log.Printf("frame size: %v", frame.Bounds())
		if err := jpeg.Encode(w, frame, nil); err != nil {
			log.Printf("error encoding frame: %v", err)
		}
	})

	// serving MJPEG is CPU and network intensive.  We split the writing and reading/encoding
	// into seperate gofuncs so we can encode the JPEGs while writing
	serveMJPEG := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boundary := "RASPIVID_mjpeg"
		w.Header().Add("Content-Type", fmt.Sprintf("multipart/x-mixed-replace;boundary=%s", boundary))
		w.Header().Add("Access-Control-Allow-Origin", "*")

		toWrite := make(chan []byte)
		var wg sync.WaitGroup
		lock := new(sync.Mutex)
		writerDone := false
		// wait for both the reader and writer
		wg.Add(2)

		// writer
		go func() {
			defer wg.Done()
		Outer:
			for {
				b, ok := <-toWrite
				if !ok {
					break
				}

				// TODO: should we check the length here?
				_, err := fmt.Fprintf(w, "\r\n--%s\r\nContent-Type: image/jpeg\r\n\r\n", boundary)
				if err != nil {
					break
				}
				// TODO: is this loop really necessary?
				for n := 0; n < len(b); {
					m, err := w.Write(b[n:])
					if err != nil {
						break Outer
					}
					n += m
				}
			}

			// this will error if the writer is actually closed, but we don't care.
			fmt.Fprintf(w, "\r\n--%s--\r\n", boundary)

			lock.Lock()
			writerDone = true
			lock.Unlock()
		}()

		// reader/encoder
		go func() {
			defer wg.Done()
			for {
				// wait to fetch the next frame
				frame, err := rawReader.WaitNextFrame()
				if err != nil {
					break
				}

				// check to see if the writer thread has finished
				isDone := false

				lock.Lock()
				isDone = writerDone
				lock.Unlock()

				if isDone {
					break
				}

				// encode to a buffer
				buf := new(bytes.Buffer)
				if err := jpeg.Encode(buf, frame, nil); err != nil {
					log.Printf("error encoding frame: %v", err)
					break
				}

				toWrite <- buf.Bytes()
			}
			close(toWrite)
		}()

		// wait for the reader and writer to finish
		wg.Wait()
	})

	mux.Handle("/frame.jpg", serveJPEG)
	mux.Handle("/frame.jpeg", serveJPEG)
	mux.HandleFunc("/frame.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/png")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if err := png.Encode(w, rawReader.Frame()); err != nil {
			log.Printf("error encoding frame: %v", err)
		}
	})
	mux.HandleFunc("/video.jpeg", serveMJPEG)
	mux.HandleFunc("/video.jpg", serveMJPEG)
	mux.HandleFunc("/frame.rgb", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		rgb := rawReader.Frame()
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
		w.Header().Add("Access-Control-Allow-Origin", "*")
		vectors := motionReader.MotionVectors()
		binary.Write(w, binary.LittleEndian, vectors)
	})
	mux.HandleFunc("/config.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")

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
