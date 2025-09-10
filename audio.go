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
	sampleDurationMs  = 40
	opusFrameDuration = time.Millisecond * sampleDurationMs
	sampleRate        = 48000
	channels          = 2
	frameSize         = sampleRate * sampleDurationMs / 1000 // 40 ms at 48kHz
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

	// Buffers
	pcm := make([]int16, frameSize*channels) // float32 PCM samples
	pcmBytes := make([]byte, len(pcm)*2)     // raw PCM bytes (2 bytes per sample)
	packet := make([]byte, 4000)             // encoded packet buffer

	close(chAudioRdy)
	for {
		// Read exactly one frame worth of PCM
		if _, err := io.ReadFull(stdout, pcmBytes); err != nil {
			checkError(&err)
		}

		// Convert byte PCM to int16 samples
		for i := range pcm {
			// Convert 2 bytes to int16 (little-endian)
			pcm[i] = int16(binary.LittleEndian.Uint16(pcmBytes[i*2:]))
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
