package rpicam

/*
#cgo LDFLAGS: -L./_build -lRpiCameraWrapper
#include "rpicam_api.h"

// Forward declaration of the Go callback
extern void goCameraCallback(char* mem, size_t size);

// Bridge function that calls the Go callback
static inline void bridgeCallback(char* mem, size_t size) {
    goCameraCallback(mem, size);
}
*/

import "C"
import (
	"unsafe"
)

var (
	rpiCamera *RpiCamera
)

type ringBuffer[T any] struct {
	ch chan T
}

func createRingBuffer[T any](size int) *ringBuffer[T] {
	return &ringBuffer[T]{
		ch: make(chan T, size),
	}
}

func (r *ringBuffer[T]) Push(data T) {
	select {
	case r.ch <- data:
	default:
		<-r.ch
		r.ch <- data
	}
}

func (r *ringBuffer[T]) Read() <-chan T {
	return r.ch
}

//export goCameraCallback
func goCameraCallback(mem *C.char, size C.size_t) {
	// Convert memory into Go []byte
	buf := C.GoBytes(unsafe.Pointer(mem), C.int(size))

	copyBuf := make([]byte, len(buf))
	copy(copyBuf, buf)

	rpiCamera.videoFeed.Push(copyBuf)
}

type RpiCamera struct {
	videoFeed *ringBuffer[[]byte]
}

func (rpiCamera *RpiCamera) StartRpiCamera() {
	ret := C.startCamera((C.CameraOutputReadyCallback)(C.bridgeCallback))
}

func CreateRpiCamera() *RpiCamera {
	rpiCamera = &RpiCamera{}
	return rpiCamera
}
