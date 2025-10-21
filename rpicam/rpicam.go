package rpicam

/*
#cgo LDFLAGS: -lRpiCameraWrapper
#include "native/rpicam_api.h"

// Forward declaration of the Go callback
extern void goCameraCallback(unsigned char* mem, size_t size);
*/
import "C"

import (
	"unsafe"

	"githug.com/pmarinov1994/baby-monitor/util"
)

var (
	rpiCamera *RpiCamera
)

//export goCameraCallback
func goCameraCallback(mem *C.uchar, size C.size_t) {
	// Convert memory into Go []byte. The result is a copy of the memory
	buf := C.GoBytes(unsafe.Pointer(mem), C.int(size))
	rpiCamera.VideoFeed.Push(buf)
}

type RpiCamera struct {
	Loglevel      uint8
	Width, Height uint32
	Framerate     uint32
	VideoFeed     *util.RingBuffer[[]byte]
}

func (rpiCamera *RpiCamera) StartRpiCamera() int {
	var params C.struct_CameraParams

	params.loglevel = C.uint8_t(rpiCamera.Loglevel)
	params.width = C.uint32_t(rpiCamera.Width)
	params.height = C.uint32_t(rpiCamera.Height)
	params.framerate = C.uint32_t(rpiCamera.Framerate)
	params.cb_yuv420 = (C.CameraOutputReadyCallback)(C.goCameraCallback)

	result := C.startCamera(&params)
	return int(result)
}

func CreateRpiCamera() *RpiCamera {
	rpiCamera = &RpiCamera{}
	rpiCamera.VideoFeed = util.CreateRingBuffer[[]byte](2)

	return rpiCamera
}
