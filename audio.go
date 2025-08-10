package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os/exec"
	"time"

	"gopkg.in/hraban/opus.v2"
)

const (
	opusFrameDuration = time.Millisecond * 20

	sampleRate = 48000
	channels   = 2
	frameSize  = 960 // 20 ms at 48kHz
)

func startAudioFeed() {
	// arecord -D hw:Zero,0 -c 2 -r 44100 -f S16_LE | ffmpeg -f s16le -ac 2 -ar 44100 -i - -ar 48000 -f s16le -
	//	arecord
	arecord := exec.Command(
		"arecord",
		// "-D", "hw:Zero,0", // TODO: handle the rpi sound card?
		"-c", fmt.Sprint(channels),
		"-r", fmt.Sprint(sampleRate),
		"-f", "FLOAT_LE", //"S16_LE",
		"-q",
	)

	stdout, err := arecord.StdoutPipe()
	if err != nil {
		panic(err)
	}

	log.Printf("starting: %s\n", arecord.String())
	if err := arecord.Start(); err != nil {
		checkError(&err)
	}

	defer arecord.Process.Kill()

	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		checkError(&err)
	}

	// Buffers
	pcm := make([]float32, frameSize*channels) // float32 PCM samples
	pcmBytes := make([]byte, len(pcm)*4)       // raw PCM bytes (4 bytes per sample)
	packet := make([]byte, 4000)               // encoded packet buffer

	for {
		// Read exactly one frame worth of PCM
		if _, err := io.ReadFull(stdout, pcmBytes); err != nil {
			checkError(&err)
		}

		// Convert byte PCM to int16 samples
		for i := range pcm {
			bits := binary.LittleEndian.Uint32(pcmBytes[i*4:])
			pcm[i] = math.Float32frombits(bits)
		}

		// Encode to Opus
		n, err := encoder.EncodeFloat32(pcm, packet)
		if err != nil {
			checkError(&err)
		}

		packet = packet[:n]
		audioFrames.Push(packet)
	}
}

func float32ToFloat64(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, f32 := range in {
		out[i] = float64(f32)
	}

	return out
}

func int32ToFloat32(in []int32) (out []float32) {
	out = make([]float32, len(in))

	for i, i32 := range in {
		out[i] = float32(i32)
	}

	return out
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, f64 := range in {
		out[i] = float32(f64)
	}

	return out
}
