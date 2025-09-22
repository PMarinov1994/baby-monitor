package rpicam

/*
#cgo LDFLAGS: -L./native/_build -lRpiCameraWrapper
#include "native/rpicam_api.h"

// Forward declaration of the Go callback
extern void goCameraCallback(char* mem, size_t size);
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
func goCameraCallback(mem *C.char, size C.size_t) {
	// Convert memory into Go []byte
	buf := C.GoBytes(unsafe.Pointer(mem), C.int(size))

	copyBuf := make([]byte, len(buf))
	copy(copyBuf, buf)

	rpiCamera.VideoFeed.Push(copyBuf)
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
