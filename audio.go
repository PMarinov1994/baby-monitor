package main

import (
	"time"

	"github.com/gordonklaus/portaudio"
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
	if err := portaudio.Initialize(); err != nil {
		checkError(&err)
	}

	defer portaudio.Terminate()

	do, err := portaudio.DefaultInputDevice()
	if err != nil {
		checkError(&err)
	}

	inParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   do,
			Channels: channels,
			Latency:  do.DefaultLowInputLatency,
		},
		SampleRate:      sampleRate,
		FramesPerBuffer: frameSize * channels,
	}

	buffer := make([]int16, frameSize*channels)
	packet := make([]byte, 4000)

	istream, err := portaudio.OpenStream(inParams, buffer)
	if err != nil {
		checkError(&err)
	}

	if err != istream.Start() {
		checkError(&err)
	}

	defer istream.Close()

	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		checkError(&err)
	}

	close(chAudioRdy)
	for {
		// Read exactly one frame worth of PCM
		if err := istream.Read(); err != nil {
			checkError(&err)
		}

		// Encode to Opus
		n, err := encoder.Encode(buffer, packet)
		if err != nil {
			checkError(&err)
		}

		packet = packet[:n]
		audioFrames.Push(packet)
	}
}
