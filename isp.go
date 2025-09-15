package main

import (
	"log"
	"os"

	"github.com/vladimirvivien/go4vl/v4l2"
)

type ISP struct {
	inFrameCh  chan []byte
	outFrameCh chan []byte

	fd uintptr

	capDev    StreamingDevice
	outputDev StreamingDevice
}

func (isp *ISP) Init(dev string, width, height uint32, outPixFmt, capPixFmt v4l2.FourCCType) error {
	var err error = nil

	isp.fd, err = v4l2.OpenDevice(dev, os.O_RDWR, 0)
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	// logCtrlInfo(isp.fd)

	outFmtMPlane, err := v4l2.GetPixFormatMPlane(isp.fd, v4l2.BufTypeVideoOutputMPlane)
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	outFmtMPlane.Width = width
	outFmtMPlane.Height = height
	outFmtMPlane.PixelFormat = outPixFmt

	if err := v4l2.SetPixFormatMPlane(isp.fd, outFmtMPlane, v4l2.BufTypeVideoOutputMPlane); err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	capFmtMplane, err := v4l2.GetPixFormatMPlane(isp.fd, v4l2.BufTypeVideoCaptureMPlane)
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	capFmtMplane.Width = width
	capFmtMplane.Height = height
	capFmtMplane.PixelFormat = capPixFmt

	if err := v4l2.SetPixFormatMPlane(isp.fd, capFmtMplane, v4l2.BufTypeVideoCaptureMPlane); err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	isp.outputDev = StreamingDevice{
		fd:      isp.fd,
		bufType: v4l2.BufTypeVideoOutputMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	outReqBuf, err := v4l2.InitBuffers(isp.outputDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	isp.outputDev.output = make(chan []byte, outReqBuf.Count)
	isp.outputDev.buffers, err = v4l2.MapMemoryBuffers(isp.outputDev) // mmap
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	isp.capDev = StreamingDevice{
		fd:      isp.fd,
		bufType: v4l2.BufTypeVideoCaptureMPlane,
		ioType:  v4l2.IOTypeMMAP,
		count:   1,
	}

	capReqBuf, err := v4l2.InitBuffers(isp.capDev) // VIDIOC_REQBUFS
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	isp.capDev.output = make(chan []byte, capReqBuf.Count)
	isp.capDev.buffers, err = v4l2.MapMemoryBuffers(isp.capDev) // mmap
	if err != nil {
		checkError(&err) // TODO: Remove
		return err
	}

	if _, err := v4l2.QueueBuffer(isp.outputDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err) // TODO: Remove
		return err
	}

	if _, err := v4l2.QueueBuffer(isp.capDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err) // TODO: Remove
		return err
	}

	if err := v4l2.StreamOn(isp.outputDev); err != nil { // VIDIOC_STREAMON
		checkError(&err) // TODO: Remove
		return err
	}

	if err := v4l2.StreamOn(isp.capDev); err != nil { // VIDIOC_STREAMON
		checkError(&err) // TODO: Remove
		return err
	}

	isp.inFrameCh = make(chan []byte)
	isp.outFrameCh = make(chan []byte)

	return nil
}

func (isp *ISP) ProcessFrame() {
	if _, err := v4l2.DequeueBuffer(isp.outputDev); err != nil { // VIDIOC_DQBUF
		checkError(&err)
	}

	frame := <-isp.inFrameCh

	n := copy(isp.outputDev.buffers[0], frame)
	if _, err := v4l2.QueueBuffer(isp.outputDev, 0, uint32(n)); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}

	convertedBuf, err := v4l2.DequeueBuffer(isp.capDev) // VIDIOC_DQBUF
	if err != nil {
		checkError(&err)
	}

	convertedFrame := make([]byte, convertedBuf.Info.Planes[0].BytesUsed)
	copy(convertedFrame, isp.capDev.buffers[0][:convertedBuf.Info.Planes[0].BytesUsed])
	isp.outFrameCh <- convertedFrame

	if _, err := v4l2.QueueBuffer(isp.capDev, 0, 0); err != nil { // VIDIOC_QBUF
		checkError(&err)
	}
}

func (isp *ISP) Close() {
	v4l2.StreamOff(isp.capDev)
	v4l2.StreamOff(isp.outputDev)

	v4l2.CloseDevice(isp.fd)
}

func (isp *ISP) LogCtrlInfo() {
	ctrls, err := v4l2.QueryAllControls(isp.fd)
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
