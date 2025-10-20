package rpicam

/*
#cgo LDFLAGS: -L/usr/lib/aarch64-linux-gnu -l:libyuv.a
#include <stdlib.h>
#include <libyuv.h>
*/
import "C"
import (
	"errors"
	"fmt"
)

// RotateI420Libyuv rotates an I420 (YUV420) frame using libyuv's SIMD code.
//
//	param: src []byte - source frame
//	param: w - width of frame
//	param: h - height of frame
//	param: hw - half the width of frame
//	param: hh - half the height of frame
//	param: angle - one of 0, 90, 180 or 270 degrees
//	returns: the rotated frame of error.
func RotateI420Libyuv(src []byte, w, h, hw, hh, angle int) ([]byte, error) {
	if len(src) != w*h*3/2 {
		return nil, fmt.Errorf("invalid I420 frame size. %d != %d", len(src), w*h*3/2)
	}

	// libyuv rotation constants
	var mode C.enum_RotationMode
	switch angle {
	case 0:
		mode = C.kRotate0
	case 90:
		mode = C.kRotate90
	case 180:
		mode = C.kRotate180
	case 270:
		mode = C.kRotate270
	default:
		return nil, errors.New("angle must be 0, 90, 180, or 270")
	}

	ySize := w * h
	uSize := hw * hh

	ySrc := &src[0]
	uSrc := &src[ySize]
	vSrc := &src[ySize+uSize]

	var dstW, dstH int
	if angle == 90 || angle == 270 {
		dstW, dstH = h, w
	} else {
		dstW, dstH = w, h
	}

	yDstSize := dstW * dstH
	uDstSize := (dstW / 2) * (dstH / 2)

	dst := make([]byte, dstW*dstH*3/2)
	yDst := &dst[0]
	uDst := &dst[yDstSize]
	vDst := &dst[yDstSize+uDstSize]

	C.I420Rotate(
		(*C.uint8_t)(ySrc), C.int(w),
		(*C.uint8_t)(uSrc), C.int(hw),
		(*C.uint8_t)(vSrc), C.int(hw),
		(*C.uint8_t)(yDst), C.int(dstW),
		(*C.uint8_t)(uDst), C.int(dstW/2),
		(*C.uint8_t)(vDst), C.int(dstW/2),
		C.int(w), C.int(h), mode,
	)

	return dst, nil
}
