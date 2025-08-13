package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"log/slog"
	"os/exec"
	"time"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/openh264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	h264FrameDuration = time.Millisecond * 20 // 50 FPS
	// h264FrameDuration = time.Millisecond * 33 // 30 FPS

	readBufferSize = 4096
	bufferSizeKB   = 256

	width     = 1920
	height    = 1080
	targetFPS = 50
)

var (
	nalSeparator = []byte{0, 0, 0, 1} //NAL break
)

type streamYUVReader struct {
	reader                   io.Reader
	width, height, frameSize int
	buf                      []byte
}

func newYUVReader(reader io.Reader, w, h int) *streamYUVReader {
	// NOTE: same as 'w*h + 2*(w/2)*(h/2)'
	sz := w * h * 3 / 2
	return &streamYUVReader{
		reader:    reader,
		width:     w,
		height:    h,
		frameSize: sz,
		buf:       make([]byte, sz),
	}
}

func (reader *streamYUVReader) Read() (image.Image, func(), error) {
	_, err := io.ReadFull(reader.reader, reader.buf)
	if err != nil {
		return nil, nil, err
	}

	yLen := reader.width * reader.height
	uLen := yLen / 4

	img := &image.YCbCr{
		Y:              reader.buf[:yLen],
		Cb:             reader.buf[yLen : yLen+uLen],
		Cr:             reader.buf[yLen+uLen:],
		YStride:        reader.width,
		CStride:        reader.width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, reader.width, reader.height),
	}

	return img, func() {}, nil
}

func startVideoFeed() {
	/*
	   0 : imx477 [4056x3040 12-bit RGGB] (/base/soc/i2c0mux/i2c@1/imx477@1a)
	       Modes: 'SRGGB10_CSI2P' : 1332x990 [120.05 fps - (696, 528)/2664x1980 crop]
	              'SRGGB12_CSI2P' : 2028x1080 [50.03 fps - (0, 440)/4056x2160 crop]
	                                2028x1520 [40.01 fps - (0, 0)/4056x3040 crop]
	                                4056x3040 [10.00 fps - (0, 0)/4056x3040 crop]
	*/
	// cmd := exec.Command("rpicam-vid", "--low-latency", "-t", "0", "--inline", "--width", "1920", "--height", "1080", "--framerate", "30", "-o", "-")
	cmd := exec.Command(
		"rpicam-vid",
		"--low-latency",
		"--flush",
		"-t", "0",
		"--inline",
		"--width", fmt.Sprint(width),
		"--height", fmt.Sprint(height),
		"--framerate", fmt.Sprint(targetFPS),
		"--codec", "yuv420",
		// "--framerate", "30",
		"-o", "-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		checkError(&err)
	}

	log.Printf("starting: %s\n", cmd.String())
	if err := cmd.Start(); err != nil {
		checkError(&err)
	}

	defer cmd.Process.Kill()

	params, err := openh264.NewParams()
	if err != nil {
		checkError(&err)
	}

	params.UsageType = openh264.CameraVideoRealTime
	params.RCMode = openh264.RCOffMode
	params.BitRate = 0 // unlimited for quality mode
	// params.IntraPeriod = targetFPS
	params.EnableFrameSkip = true
	params.IntraPeriod = 30
	params.SliceNum = 1          // Defaults to single NAL unit mode
	params.MultipleThreadIdc = 8 // TODO:
	params.MaxNalSize = 0
	params.SliceMode = openh264.SMFixedslcnumSlice
	params.SliceSizeConstraint = 12800 * 5

	reader := newYUVReader(stdout, width, height)

	mediaProps := prop.Media{
		Video: prop.Video{
			Width:       width,
			Height:      height,
			FrameRate:   float32(targetFPS),
			FrameFormat: frame.FormatI420,
		},
	}

	encoder, err := params.BuildVideoEncoder(reader, mediaProps)
	if err != nil {
		checkError(&err)
	}

	defer encoder.Close()

	proccessVideoFeed(encoder)
}

// NOTE: from https://github.com/bezineb5/go-h264-streamer/blob/main/stream/streaming.go
func proccessVideoFeed(videoFeed codec.ReadCloser) {
	nalBuf := make([]byte, bufferSizeKB*1024)
	currentPos := 0
	NALlen := len(nalSeparator)

	for {
		inBuf, _, err := videoFeed.Read()
		if err != nil {
			if err == io.EOF {
				slog.Debug("startCamera: EOF", slog.String("command", "rpicam-vid"))
				return
			}
			slog.Error("startCamera: Error reading from camera; ignoring", slog.Any("error", err))
			continue
		}

		copied := copy(nalBuf[currentPos:], inBuf)
		startPosSearch := currentPos - NALlen
		endPos := currentPos + copied

		if startPosSearch < 0 {
			startPosSearch = 0
		}
		nalIndex := bytes.Index(nalBuf[startPosSearch:endPos], nalSeparator)

		currentPos = endPos
		if nalIndex > 0 {
			nalIndex += startPosSearch

			// Boadcast before the NAL
			broadcast := make([]byte, nalIndex)
			copy(broadcast, nalBuf)
			videoFrames.Push(broadcast)

			// Shift
			copy(nalBuf, nalBuf[nalIndex:currentPos])
			currentPos = currentPos - nalIndex
		}
	}
}

func isVideoSourceAvailable() bool {
	rpicam := exec.Command(
		"rpicam-vid",
		"--version",
	)

	if err := rpicam.Start(); err != nil {
		return false // executable not found on $PATH
	}

	state, err := rpicam.Process.Wait()
	if err != nil {
		checkError(&err)
	}

	return state.ExitCode() == 0
}
