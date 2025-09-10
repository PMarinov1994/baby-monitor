package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"gopkg.in/hraban/opus.v2"
)

const (
	sampleRate = 48000
	channels   = 2
	// frameSize  = 960 // 20 ms at 48kHz
	frameSize         = sampleRate * 40 / 1000 // 20 ms at 48kHz
	bitsPerSample     = 16                     // 16-bit audio
	opusFrameDuration = time.Millisecond * 40
)

var (
	chAudioRdy = make(chan struct{})
)

func startAudioFeed() {
	// arecord -D hw:Zero,0 -c 2 -r 44100 -f S16_LE | ffmpeg -f s16le -ac 2 -ar 44100 -i - -ar 48000 -f s16le -
	//	arecord
	arecord := exec.Command(
		"arecord",
		// "-D", "hw:Zero,0", // TODO: handle the rpi sound card?
		"-c", fmt.Sprint(channels),
		"-r", fmt.Sprint(sampleRate),
		"-f", "S16_LE",
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

	// Calculate bytes per chunk
	samplesPerChunk := int(float64(sampleRate) * opusFrameDuration.Seconds())
	bytesPerSample := bitsPerSample / 8
	chunkSize := samplesPerChunk * channels * bytesPerSample // 7680 bytes for 40ms at 48kHz, 16-bit, stereo
	totalSamplesPerChunk := samplesPerChunk * channels       // 3840 samples (interleaved)

	// Buffers
	pcm := make([]int16, totalSamplesPerChunk) // float32 PCM samples
	pcmBytes := make([]byte, chunkSize)        // raw PCM bytes (4 bytes per sample)
	packet := make([]byte, 4000)               // encoded packet buffer

	close(chAudioRdy)
	for {
		// Read exactly one frame worth of PCM
		if _, err := io.ReadFull(stdout, pcmBytes); err != nil {
			checkError(&err)
		}

		// Convert byte PCM to int16 samples
		for i := range totalSamplesPerChunk {
			// Convert 2 bytes to int16 (little-endian)
			pcm[i] = int16(binary.LittleEndian.Uint16(pcmBytes[i*2 : (i+1)*2]))
		}

		// Encode to Opus
		n, err := encoder.Encode(pcm, packet)
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
