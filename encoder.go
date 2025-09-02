package main

import (
	"log"
	"os"

	"github.com/vladimirvivien/go4vl/v4l2"
)

const (
	H264_MINIMUM_QP_VALUE = 0x00990A61
	H264_MAXIMUM_QP_VALUE = 0x00990A62
	H264_I_FRAME_PERIOD   = 0x00990A66
	H264_PROFILE          = 0x00990A6B
)

type Encoder struct {
	rawFrameCh     chan []byte
	encodedFrameCh chan []byte

	fd uintptr

	capDev    StreamingDevice
	outputDev StreamingDevice
}

func (encoder *Encoder) Init(dev string, width, height uint32) {
	var err error = nil

	encoder.fd, err = v4l2.OpenDevice("/dev/video11", os.O_RDWR, 0)
	if err != nil {
		checkError(&err)
	}

	logCtrlInfo(encoder.fd)

	// NOTE: This is important for live streams, since clients can join at any time,
	//       and we need to let them know about the h264 SPS/PPS (sequence) parameters
	if err := v4l2.SetControlValue(encoder.fd, v4l2.CtrlMpegRepeatSeqHeader, 1); err != nil {
		checkError(&err)
	}

	outFmMplane, err := v4l2.GetPixFormatMPlane(encoder.fd, v4l2.BufTypeVideoOutputMPlane)
	if err != nil {
		checkError(&err)
	}

	outFmMplane.Width = width
	outFmMplane.Height = height
	outFmMplane.PixelFormat = v4l2.PixelFmtYUV410

	if err := v4l2.SetPixFormatMPlane(encoder.fd, outFmMplane, v4l2.BufTypeVideoOutputMPlane); err != nil {
		checkError(&err)
	}

	capFmMplane, err := v4l2.GetPixFormatMPlane(encoder.fd, v4l2.BufTypeVideoCaptureMPlane)
	if err != nil {
		checkError(&err)
	}

	capFmMplane.Width = width
	capFmMplane.Height = height

	if err := v4l2.SetPixFormatMPlane(encoder.fd, capFmMplane, v4l2.BufTypeVideoCaptureMPlane); err != nil {
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

	if err := v4l2.SetStreamParam(encoder.fd, v4l2.BufTypeVideoOutputMPlane, streamParam); err != nil {
		checkError(&err)
	}

	encoder.outputDev = StreamingDevice{
		fd:      encoder.fd,
		bufType: v4l2.BufTypeVideoOutputMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	outReqBuf, err := v4l2.InitBuffers(encoder.outputDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err)
	}

	encoder.outputDev.output = make(chan []byte, outReqBuf.Count)
	encoder.outputDev.buffers, err = v4l2.MapMemoryBuffers(encoder.outputDev) // mmap
	if err != nil {
		checkError(&err)
	}

	encoder.capDev = StreamingDevice{
		fd:      encoder.fd,
		bufType: v4l2.BufTypeVideoCaptureMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	capReqBuf, err := v4l2.InitBuffers(encoder.capDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err)
	}

	encoder.capDev.output = make(chan []byte, capReqBuf.Count)
	encoder.capDev.buffers, err = v4l2.MapMemoryBuffers(encoder.capDev) // mmap
	if err != nil {
		checkError(&err)
	}

	if _, err := v4l2.QueueBuffer(encoder.outputDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	if _, err := v4l2.QueueBuffer(encoder.capDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	if err := v4l2.StreamOn(encoder.outputDev); err != nil { // VIDIOC_STREAMON
		checkError(&err)
	}

	if err := v4l2.StreamOn(encoder.capDev); err != nil { // VIDIOC_STREAMON
		checkError(&err)
	}

	encoder.rawFrameCh = make(chan []byte)
	encoder.encodedFrameCh = make(chan []byte)
}

func (encoder *Encoder) ProcessFrame() {
	if _, err := v4l2.DequeueBuffer(encoder.outputDev); err != nil { // VIDIOC_DQBUF
		checkError(&err)
	}

	frame := <-encoder.rawFrameCh

	copy(encoder.outputDev.buffers[0], frame)
	if _, err := v4l2.QueueBuffer(encoder.outputDev, 0, uint32(len(frame))); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	encodedBuf, err := v4l2.DequeueBuffer(encoder.capDev) // VIDIOC_DQBUF
	if err != nil {
		checkError(&err)
	}

	encodedFrame := make([]byte, encodedBuf.Info.Planes[0].BytesUsed)
	copy(encodedFrame, encoder.capDev.buffers[0][:encodedBuf.Info.Planes[0].BytesUsed])
	encoder.encodedFrameCh <- encodedFrame

	if _, err := v4l2.QueueBuffer(encoder.capDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}
}

func (encoder *Encoder) Close() {
	v4l2.StreamOff(encoder.capDev)
	v4l2.StreamOff(encoder.outputDev)

	v4l2.CloseDevice(encoder.fd)
}

func (encoder *Encoder) LogCtrlInfo() {
	ctrls, err := v4l2.QueryAllControls(encoder.fd)
	if err != nil {
		checkError(&err)
	}

	log.Println("-> Controls")
	for _, ctrl := range ctrls {
		log.Printf("--> %#v\n", ctrl)
	}

	// extCtrls, err := v4l2.QueryAllExtControls(encoder.fd)
	// if err != nil {
	// 	checkError(&err)
	// }
	//
	// log.Println("-> ExtControls")
	// for _, ctrl := range extCtrls {
	// 	log.Printf("--> %#v\n", ctrl)
	// }
}
