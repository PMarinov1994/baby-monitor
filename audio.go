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
	opusFrameDuration = time.Millisecond * 20

	inSampleRate  = 44100
	outSampleRate = 48000
	channels      = 2
	frameSize     = 960 // 20 ms at 48kHz

	useFFMPEG = false
)

func startAudioFeed() {
	// arecord -D hw:Zero,0 -c 2 -r 44100 -f S16_LE | ffmpeg -f s16le -ac 2 -ar 44100 -i - -ar 48000 -f s16le -
	//	arecord
	arecord := exec.Command(
		"arecord",
		// "-D", "hw:Zero,0", // TODO: handle the rpi sound card?
		"-c", fmt.Sprint(channels),
		"-r", fmt.Sprint(outSampleRate),
		"-f", "S16_LE",
		"-q",
	)

	arecordOut, err := arecord.StdoutPipe()
	if err != nil {
		panic(err)
	}

	var (
		stdout io.ReadCloser
		ffmpeg *exec.Cmd
	)

	if useFFMPEG {
		ffmpeg = exec.Command(
			"ffmpeg",
			"-f", "s16le",
			"-ac", fmt.Sprint(channels),
			"-ar", fmt.Sprint(inSampleRate),
			"-i", "-",
			"-loglevel", "quiet",
			"-ar", fmt.Sprint(outSampleRate),
			"-flags", "low_delay",
			"-fflags", "nobuffer",
			"-f", "s16le",
			"-filter:a", "volume=100",
			"-",
		)

		ffmpeg.Stdin = arecordOut

		stdout, err = ffmpeg.StdoutPipe()
		if err != nil {
			panic(err)
		}
	} else {
		stdout = arecordOut
	}

	log.Printf("starting: %s\n", arecord.String())
	if err := arecord.Start(); err != nil {
		checkError(&err)
	}

	defer arecord.Process.Kill()

	if useFFMPEG {
		log.Printf("starting: %s\n", ffmpeg.String())
		if err := ffmpeg.Start(); err != nil {
			checkError(&err)
		}

		defer ffmpeg.Process.Kill()
	}

	encoder, err := opus.NewEncoder(outSampleRate, channels, opus.AppAudio)
	if err != nil {
		checkError(&err)
	}

	// Buffers
	pcm := make([]int16, frameSize*channels) // int16 PCM samples
	pcmBytes := make([]byte, len(pcm)*2)     // raw PCM bytes (2 bytes per sample)
	packet := make([]byte, 4000)             // encoded packet buffer

	for {
		// Read exactly one frame worth of PCM
		if _, err := io.ReadFull(stdout, pcmBytes); err != nil {
			checkError(&err)
		}

		// Convert byte PCM to int16 samples
		for i := range pcm {
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
