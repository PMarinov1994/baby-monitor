package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/pmarinov1994/go4vl/v4l2"
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
		"--low-latency",
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

	fd, err := v4l2.OpenDevice("/dev/video11", os.O_RDWR, 0)
	if err != nil {
		checkError(&err)
	}

	defer v4l2.CloseDevice(fd)

	logCtrlInfo(fd)

	// NOTE: This is important for live streams, since clients can join at any time,
	//       and we need to let them know about the h264 SPS/PPS (sequence) parameters
	if err := v4l2.SetControlValue(fd, v4l2.CtrlMpegRepeatSeqHeader, 1); err != nil {
		checkError(&err)
	}

	outFmMplane, err := v4l2.GetPixFormatMPlane(fd, v4l2.BufTypeVideoOutputMPlane)
	if err != nil {
		checkError(&err)
	}

	outFmMplane.Width = width
	outFmMplane.Height = height
	outFmMplane.PixelFormat = v4l2.PixelFmtYUV410

	if err := v4l2.SetPixFormatMPlane(fd, outFmMplane, v4l2.BufTypeVideoOutputMPlane); err != nil {
		checkError(&err)
	}

	capFmMplane, err := v4l2.GetPixFormatMPlane(fd, v4l2.BufTypeVideoCaptureMPlane)
	if err != nil {
		checkError(&err)
	}

	capFmMplane.Width = width
	capFmMplane.Height = height

	if err := v4l2.SetPixFormatMPlane(fd, capFmMplane, v4l2.BufTypeVideoCaptureMPlane); err != nil {
		checkError(&err)
	}

	streamParam := v4l2.StreamParam{
		Type: v4l2.BufTypeVideoOutputMPlane,
		Output: v4l2.OutputParam{
			TimePerFrame: v4l2.Fract{
				Numerator:   1,
				Denominator: 30,
			},
		},
	}

	if err := v4l2.SetStreamParam(fd, v4l2.BufTypeVideoOutputMPlane, streamParam); err != nil {
		checkError(&err)
	}

	outputDev := StreamingDevice{
		fd:      fd,
		bufType: v4l2.BufTypeVideoOutputMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	outReqBuf, err := v4l2.InitBuffers(outputDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err)
	}

	outputDev.output = make(chan []byte, outReqBuf.Count)
	outputDev.buffers, err = v4l2.MapMemoryBuffers(outputDev) // mmap
	if err != nil {
		checkError(&err)
	}

	capDev := StreamingDevice{
		fd:      fd,
		bufType: v4l2.BufTypeVideoCaptureMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	capReqBuf, err := v4l2.InitBuffers(capDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err)
	}

	capDev.output = make(chan []byte, capReqBuf.Count)
	capDev.buffers, err = v4l2.MapMemoryBuffers(capDev) // mmap
	if err != nil {
		checkError(&err)
	}

	if _, err := v4l2.QueueBuffer(outputDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	if _, err := v4l2.QueueBuffer(capDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	if err := v4l2.StreamOn(outputDev); err != nil { // VIDIOC_STREAMON
		checkError(&err)
	}

	defer v4l2.StreamOff(outputDev)

	if err := v4l2.StreamOn(capDev); err != nil { // VIDIOC_STREAMON
		checkError(&err)
	}

	defer v4l2.StreamOff(capDev)

	defer func() {
		log.Printf("Closing stuf")
	}()

	// TODO: To low of a value and no frames are visible
	outCh := createRingBuffer[[]byte](512)

	go proccessVideoFeed(outCh)
	for {

		if _, err := v4l2.DequeueBuffer(outputDev); err != nil { // VIDIOC_DQBUF
			checkError(&err)
		}

		frame, err := camReader.Read()
		if err != nil {
			checkError(&err)
		}

		copy(outputDev.buffers[0], frame)
		if _, err := v4l2.QueueBuffer(outputDev, 0, uint32(len(frame))); err != nil { // VIDIOC_QBUF
			checkError(&err)
		}

		encodedBuf, err := v4l2.DequeueBuffer(capDev) // VIDIOC_DQBUF
		if err != nil {
			checkError(&err)
		}

		encodedFrame := make([]byte, encodedBuf.Info.Planes[0].BytesUsed)
		copy(encodedFrame, capDev.buffers[0][:encodedBuf.Info.Planes[0].BytesUsed])
		outCh.Push(encodedFrame)

		if _, err := v4l2.QueueBuffer(capDev, 0, 0); err != nil { // VIDIOC_QBUF
			checkError(&err)
		}
	}
}

// NOTE: from https://github.com/bezineb5/go-h264-streamer/blob/main/stream/streaming.go
func proccessVideoFeed(videoFeed *ringBuffer[[]byte]) {
	nalBuf := make([]byte, bufferSizeKB*1024)
	currentPos := 0
	NALlen := len(nalSeparator)

	for {
		inBuf := <-videoFeed.Read()

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
