package main

import (
	"context"
	"encoding/binary"

	empty "github.com/golang/protobuf/ptypes/empty"

	"bytes"
	"image/jpeg"
	"log"

	"github.com/donniet/raspividWrapper/videoService"
)

/*
VideoServerGRPC is a GRPC server for frames and motion vectors
*/
type VideoServerGRPC struct {
	MotionReader *MotionVectorReader
	VideoReader  *RawVideoReader
}

/*
MotionRaw streams byte arras of the raw motion vectors via GRPC
*/
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

/*
FrameJPEG returns the byte array of a JPEG of a single frame
*/
func (v *VideoServerGRPC) FrameJPEG(ctx context.Context, _ *empty.Empty) (*videoService.Frame, error) {
	frame := v.VideoReader.Frame()

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, frame, &jpeg.Options{Quality: jpegQuality}); err != nil {
		log.Fatalf("error encoding JPEG to buffer: %v", err)
	}

	return &videoService.Frame{Data: buf.Bytes()}, nil
}

/*
VideoJPEG streams byte arrays of JPEGs of frames from the video
*/
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

/*
FrameRaw returns the raw RGB24 (w,h,c) of the current frame
*/
func (v *VideoServerGRPC) FrameRaw(ctx context.Context, _ *empty.Empty) (*videoService.Frame, error) {
	rgb := v.VideoReader.Frame()

	return &videoService.Frame{Data: rgb.Pix}, nil
}

/*
VideoRaw streams the raw RGB24 (w,h,c) as byte arrays
*/
func (v *VideoServerGRPC) VideoRaw(_ *empty.Empty, rawServer videoService.Video_VideoRawServer) error {
	for {
		rgb, err := v.VideoReader.WaitNextFrame()

		if err != nil {
			break
		}

		rawServer.Send(&videoService.Frame{Data: rgb.Pix})
	}

	return nil
}

/*
MetaData returns the settings of the video and motion vectors via GRPC
*/
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
