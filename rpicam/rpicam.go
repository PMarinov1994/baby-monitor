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
	VideoFeed *util.RingBuffer[[]byte]
}

func (rpiCamera *RpiCamera) StartRpiCamera() {
	C.startCamera((C.CameraOutputReadyCallback)(C.goCameraCallback))
}

func CreateRpiCamera() *RpiCamera {
	rpiCamera = &RpiCamera{}
	rpiCamera.VideoFeed = util.CreateRingBuffer[[]byte](1)

	return rpiCamera
}
