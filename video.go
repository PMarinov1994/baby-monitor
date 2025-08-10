package main

import (
	"bytes"
	"io"
	"log"
	"log/slog"
	"os/exec"
	"time"
)

const (
	h264FrameDuration = time.Millisecond * 20 // 50 FPS
	// h264FrameDuration = time.Millisecond * 33 // 30 FPS

	readBufferSize = 4096
	bufferSizeKB   = 256
)

var (
	nalSeparator = []byte{0, 0, 0, 1} //NAL break
)

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
		"--inline",
		"--width", "2028",
		"--height", "1080",
		"--framerate", "50",
		// "--framerate", "30",
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

	proccessVideoFeed(stdout)
}

// NOTE: from https://github.com/bezineb5/go-h264-streamer/blob/main/stream/streaming.go
func proccessVideoFeed(videoFeed io.Reader) {
	p := make([]byte, readBufferSize)
	buffer := make([]byte, bufferSizeKB*1024)
	currentPos := 0
	NALlen := len(nalSeparator)

	for {
		n, err := videoFeed.Read(p)
		if err != nil {
			if err == io.EOF {
				slog.Debug("startCamera: EOF", slog.String("command", "rpicam-vid"))
				return
			}
			slog.Error("startCamera: Error reading from camera; ignoring", slog.Any("error", err))
			continue
		}

		copied := copy(buffer[currentPos:], p[:n])
		startPosSearch := currentPos - NALlen
		endPos := currentPos + copied

		if startPosSearch < 0 {
			startPosSearch = 0
		}
		nalIndex := bytes.Index(buffer[startPosSearch:endPos], nalSeparator)

		currentPos = endPos
		if nalIndex > 0 {
			nalIndex += startPosSearch

			// Boadcast before the NAL
			broadcast := make([]byte, nalIndex)
			copy(broadcast, buffer)
			videoFrames.Push(broadcast)

			// Shift
			copy(buffer, buffer[nalIndex:currentPos])
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
