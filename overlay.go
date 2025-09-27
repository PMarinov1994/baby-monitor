package main

import (
	"fmt"

	"githug.com/pmarinov1994/baby-monitor/util"
)

type OverlayData struct {
	x, y       int16
	uVal, vVal int8
}

const (
	// The width of the overlay is calculated based on the picture width
	// divided by this number
	overlayWidthDivider = 2

	// Overlay pixel size will take this much picture pixels
	pixelSize = 3

	// The amount of acceptable overlay updates per frame
	// that we are ok to be skipped. Any more than this number
	// and we will panic, terminating the program.
	overlayUpdateSkips = 100
)

var (
	// NOTE: Race condition will hapen at some point, if not enough data is fed
	//       to the channel.
	dbPerFrame = util.CreateRingBuffer[float64](4)

	dbFrameOverlay     = make([]float64, width/overlayWidthDivider)
	dbFrameOverlayHead = 0

	skippedOverlay uint64 = 0
)

func applyOverlay(frame *[]byte) {
	select {
	case db := <-dbPerFrame.Read():
		dbFrameOverlay[dbFrameOverlayHead] = db
		dbFrameOverlayHead++

		if dbFrameOverlayHead == len(dbFrameOverlay) {
			dbFrameOverlayHead = 0
		}

		skippedOverlay = 0
	default:
		skippedOverlay++
	}

	if skippedOverlay > overlayUpdateSkips {
		panic(fmt.Sprintf("Skipped overlay new data: %d\n", skippedOverlay))
	}

	lastX, lastY := 0, 0

	// Scuffed do ... while(...) {...}
	firstLoop := true
	for x, i := 0, dbFrameOverlayHead; ; x, i = x+1, i+1 {
		if i == len(dbFrameOverlay) {
			i = 0
		}

		if !firstLoop && i == dbFrameOverlayHead {
			break
		}

		db := dbFrameOverlay[i]
		y := hheight - int(db)

		drawLine(frame, lastX, lastY, x, y, width, height, hwidth, hheight)
		lastX = x
		lastY = y

		// We are no longer in first loop
		firstLoop = false
	}
}

func drawLine(frame *[]byte, fromX, fromY, toX, toY, w, h, hw, hh int) {
	xDir := 1
	if fromX > toX {
		xDir = -1
	}

	yDir := 1
	if fromY > toY {
		yDir = -1
	}

	for x, y := fromX, fromY; x != toX || y != toY; {
		drawDot(frame, x, y, pixelSize, w, h, hw, hh)

		if x != toX {
			x += xDir
		}

		if y != toY {
			y += yDir
		}
	}
}

func drawDot(frame *[]byte, x, y, pixSize, w, h, hw, hh int) {
	for i := x; i < (x + pixSize); i++ {
		for j := y; j < (y + pixSize); j++ {
			setPixelYUV420(frame, i, j, w, h, hw, hh)
		}
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
