package main

import (
	"githug.com/pmarinov1994/baby-monitor/util"
)

type OverlayData struct {
	x, y       int16
	uVal, vVal int8
}

const (
	pixelBin = 4
)

var (
	dbPerFrame = util.CreateRingBuffer[float64](1)

	dbFrameOverlay     = make([]float64, width/pixelBin)
	dbFrameOverlayHead = 0
)

func applyOverlay(frame *[]byte) {
	select {
	case db := <-dbPerFrame.Read():
		dbFrameOverlay[dbFrameOverlayHead] = db
		dbFrameOverlayHead++

		if dbFrameOverlayHead == len(dbFrameOverlay) {
			dbFrameOverlayHead = 0
		}
	default:
	}

	// Scufed do ... while(...) {...}
	firstLoop := true
	for x, i := 0, dbFrameOverlayHead; ; x, i = x+(pixelBin/2), i+1 {
		if i == len(dbFrameOverlay) {
			i = 0
		}

		if !firstLoop && i == dbFrameOverlayHead {
			break
		}

		db := dbFrameOverlay[i]
		y := hheight - int(db)

		for j := x; j < x+pixelBin; j++ {
			for k := y - 1; k < y-1+pixelBin; k++ {
				setPixelYUV420(frame, j, k, width, height, hwidth, hheight)
			}
		}

		// We are no longer in first loop
		firstLoop = false
	}
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
