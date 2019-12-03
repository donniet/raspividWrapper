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
	"time"

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
	framerate    = 30
	keyFrameRate = framerate * 2
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
	// math taken from here https://github.com/billw2/pikrellcam/blob/master/src/motion.c#L1634
	motionReader := &MotionVectorReader{
		MBX: 1 + (width+15)/16,
		MBY: 1 + height/16,
	}
	go func() {
		if err := motionReader.Start(motionPipe); err != nil {
			log.Printf("motionReader start error: %v", err)
		}
	}()
	defer motionReader.Close()

	// starting the raw reader
	stride := 3 * width
	if r := stride % 16; r != 0 {
		stride += 3 * r
	}
	rawReader := &RawVideoReader{
		Stride: stride,
		Cols:   width,
		Rows:   height,
	}
	go func() {
		if err := rawReader.Start(rawPipe); err != nil {
			log.Printf("rawReader Start error: %v", err)
		}
	}()
	defer rawReader.Close()

	// starting the socket server
	sock := NewSocketServer()
	go func() {
		if err := sock.ServeTCP(videoPort); err != nil {
			log.Printf("error serving TCP: %v", err)
		}
	}()
	defer sock.Close()

	// write directly from the video pipe to all listening sockets
	// TODO: should we attempt to parse the h264 stream at all?
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

	// create an HTTP server for frames and metadata
	mux := http.NewServeMux()
	serveJPEG := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "image/jpeg")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		frame := rawReader.Frame()
		// log.Printf("frame size: %v", frame.Bounds())
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
		// this channel allows communication to the reader to stop if the writer fails
		isDone := make(chan bool)
		// wait for both the reader and writer
		wg.Add(2)

		// writer
		go func() {
			defer wg.Done()
			for {
				b, ok := <-toWrite
				if !ok {
					break
				}

				_, err := fmt.Fprintf(w, "\r\n--%s\r\nContent-Type: image/jpeg\r\n\r\n", boundary)
				if err != nil {
					break
				}

				// Writer must return a non-nil error if the bytes written is less than the len(b)
				if _, err := w.Write(b); err != nil {
					break
				}
			}

			// this will error if the writer is actually closed, but we don't care.
			fmt.Fprintf(w, "\r\n--%s--\r\n", boundary)

			close(isDone)
		}()

		// reader/encoder
		go func() {
			defer wg.Done()
		Outer:
			for {
				// wait to fetch the next frame
				frame, err := rawReader.WaitNextFrame()
				if err != nil {
					break
				}

				// if the isDone channel is closed we will break from the outer loop,
				// otherwise continue
				select {
				case <-isDone:
					break Outer
				default:
				}

				// encode to a buffer
				buf := new(bytes.Buffer)
				if err := jpeg.Encode(buf, frame, nil); err != nil {
					log.Printf("error encoding frame: %v", err)
					break
				}
				// slow down the framerate
				time.Sleep(60 * time.Millisecond)

				toWrite <- buf.Bytes()
			}
			close(toWrite)
		}()

		// wait for the reader and writer to finish before returning from the handlerfunc
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
			"mbx":          motionReader.MBX,
			"mby":          motionReader.MBY,
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

	// start a GRPC server for frames and metadata to avoid overhead of HTTP
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

	// wait for either our process to be interrupted, or the wrapped process to end.
	// TODO: what other conditions should cause this process to fail?
	select {
	case <-interrupted:
	case <-processEnded:
	}

	fmt.Println("Closing")
}
