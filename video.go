package main

import (
	"io"
	"log"
	"time"

	"github.com/vladimirvivien/go4vl/v4l2"
	"githug.com/pmarinov1994/baby-monitor/rpicam"
	"githug.com/pmarinov1994/baby-monitor/util"
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

	hwidth  = width / 2
	hheight = height / 2
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

	rpiCam := rpicam.CreateRpiCamera()
	rpiCam.Width = width
	rpiCam.Height = height
	rpiCam.Framerate = targetFPS
	rpiCam.Loglevel = 2

	go func() {
		// TODO: We need to shutdown this native loop
		rpiCam.StartRpiCamera()
	}()

	var encoder Encoder
	if err := encoder.Init(
		"/dev/video11",
		width,
		height,
		v4l2.PixelFmtYUV420); err != nil {
		util.CheckError(&err)
	}

	go func() {
		for {
			if shutdown {
				return
			}

			encoder.ProcessFrame()
		}
	}()

	close(chVideoRdy)

	for rawFrame := range rpiCam.VideoFeed.Read() {
		if shutdown {
			return
		}

		// Apply overlay
		applyOverlay(&rawFrame)

		// Feed the encoder
		encoder.rawFrameCh <- rawFrame
		// Get frame
		encodedFrame := <-encoder.encodedFrameCh

		// Push frame to packetizer
		videoFrames.Push(encodedFrame)
	}
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
		util.CheckError(&err)
	}

	log.Println("-> ExtControls")
	for _, ctrl := range extCtrls {
		log.Printf("--> %#v\n", ctrl)
	}
}
