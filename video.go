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
	// "github.com/pion/mediadevices/pkg/codec/mmal"
	"github.com/pion/mediadevices/pkg/codec/openh264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	// h264FrameDuration = time.Millisecond * 20 // 50 FPS
	h264FrameDuration = time.Millisecond * 33 // 30 FPS

	readBufferSize = 4096
	bufferSizeKB   = 256

	// FullHD
	// width     = 1920
	// height    = 1080

	// HD
	width  = 1280
	height = 720

	targetFPS = 30
	// targetFPS = 50
)

var (
	nalSeparator = []byte{0, 0, 0, 1} //NAL break
)

type streamYUVReader struct {
	reader                io.Reader
	width, height         int
	frameSize             int
	halfWidth, halfHeight int
	buf                   []byte
}

func newYUVReader(reader io.Reader, w, h int) *streamYUVReader {
	// NOTE: same as 'w*h + 2*(w/2)*(h/2)'
	sz := w * h * 3 / 2
	return &streamYUVReader{
		reader:     reader,
		width:      w,
		halfWidth:  w / 2,
		height:     h,
		halfHeight: h / 2,
		frameSize:  sz,
		buf:        make([]byte, sz),
	}
}

func (reader *streamYUVReader) Read() (image.Image, func(), error) {
	_, err := io.ReadFull(reader.reader, reader.buf)
	if err != nil {
		return nil, nil, err
	}

	yLen := reader.width * reader.height
	uLen := reader.halfWidth * reader.halfHeight // uLen := yLen / 4

	if false { // TODO: test after fixing encoding
		centerX := width / 2
		centerY := height / 2
		for y := centerY - 3; y <= centerY+3; y++ {
			for x := centerX - 3; x <= centerX+3; x++ {
				setPixelYUV420(&reader.buf,
					x, y,
					reader.width,
					reader.height,
					reader.halfWidth,
					reader.halfHeight)
			}
		}
	}

	img := &image.YCbCr{
		Y:              reader.buf[:yLen],
		Cb:             reader.buf[yLen : yLen+uLen],
		Cr:             reader.buf[yLen+uLen:],
		YStride:        reader.width,
		CStride:        reader.halfWidth,
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

	var encoder codec.ReadCloser

	reader := newYUVReader(stdout, width, height)
	mediaProps := prop.Media{
		Video: prop.Video{
			Width:       width,
			Height:      height,
			FrameRate:   float32(targetFPS),
			FrameFormat: frame.FormatI420,
		},
	}

	if false { // TODO: Check based on encoding (hardware vs software)
		// params, err := mmal.NewParams()
		// if err != nil {
		// 	checkError(&err)
		// }
		//
		// params.BitRate = 5_000_000
		// params.KeyFrameInterval = 30
		//
		// encoder, err = params.BuildVideoEncoder(reader, mediaProps)
		// if err != nil {
		// 	checkError(&err)
		// }
	} else {
		params, err := openh264.NewParams()

		if err != nil {

			checkError(&err)

		}

		params.UsageType = openh264.CameraVideoRealTime
		params.RCMode = openh264.RCBitrateMode
		params.BitRate = 5_000_000
		// params.IntraPeriod = targetFPS
		params.EnableFrameSkip = true
		params.IntraPeriod = 30
		params.MultipleThreadIdc = 1 // TODO:
		params.MaxNalSize = 0
		params.SliceNum = 1 // Defaults to single NAL unit mode
		params.SliceMode = openh264.SMSizelimitedSlice
		params.SliceSizeConstraint = 12800 * 5

		encoder, err = params.BuildVideoEncoder(reader, mediaProps)
		if err != nil {
			checkError(&err)
		}
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

// SetPixelYUV420 sets the pixel at (x, y) to black in a YUV420 planar buffer.
// The frame is modified in-place through the pointer to the byte slice.
func setPixelYUV420(frame *[]byte, x, y, width, hight, hwidth, hhight int) {
	if x < 0 || x >= width || y < 0 || y >= height {
		return // out of bounds
	}

	yPlaneSize := width * height
	uvPlaneSize := (hwidth) * (hhight)

	buf := *frame

	// Set Y (luma) to black
	yIndex := y*width + x
	buf[yIndex] = 0

	// Set U and V (chroma) to neutral (128)
	uIndex := yPlaneSize + (y/2)*(width/2) + (x / 2)
	vIndex := yPlaneSize + uvPlaneSize + (y/2)*(hwidth) + (x / 2)
	buf[uIndex] = 128
	buf[vIndex] = 128
}
