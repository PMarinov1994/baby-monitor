package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/vladimirvivien/go4vl/v4l2"
)

const (
	// FullHD
	// width     = 1920
	// height    = 1080

	// 2028x1080
	// width  = 2028
	// height = 1080

	// HD
	width  = 1280
	height = 720

	targetFPS = 25

	h264FrameDuration = time.Duration(time.Second / targetFPS)
)

var (
	chVideoRdy = make(chan struct{})
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

func (reader *streamYUVReader) Read() ([]byte, error) {
	_, err := io.ReadFull(reader.reader, reader.buf)
	if err != nil {
		return nil, err
	}

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

	return reader.buf, nil
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
		"--nopreview",
		"--flush",
		"-t", "0",
		"--width", fmt.Sprint(width),
		"--height", fmt.Sprint(height),
		"--framerate", fmt.Sprint(targetFPS),
		"--codec", "yuv420",
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

	camReader := newYUVReader(stdout, width, height)

	var encoder Encoder
	if err := encoder.Init("/dev/video11", width, height); err != nil {
		checkError(&err)
	}

	go func() {
		for {
			encoder.ProcessFrame()
		}
	}()

	defer encoder.Close()

	close(chVideoRdy)
	for {

		frame, err := camReader.Read()
		if err != nil {
			checkError(&err)
		}

		// Feed the encoder
		encoder.rawFrameCh <- frame
		// Get frame
		videoFrames.Push(<-encoder.encodedFrameCh)
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
func setPixelYUV420(frame *[]byte, x, y, width, height, hwidth, hhight int) {
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

func logCtrlInfo(fd uintptr) {
	// ctrls, err := v4l2.QueryAllControls(fd)
	// if err != nil {
	// 	checkError(&err)
	// }
	//
	// log.Println("-> Controls")
	// for _, ctrl := range ctrls {
	// 	log.Printf("--> %#v\n", ctrl)
	// }

	extCtrls, err := v4l2.QueryAllExtControls(fd)
	if err != nil {
		checkError(&err)
	}

	log.Println("-> ExtControls")
	for _, ctrl := range extCtrls {
		log.Printf("--> %#v\n", ctrl)
	}
}
