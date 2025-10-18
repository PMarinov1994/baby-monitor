package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/gordonklaus/portaudio"
	"githug.com/pmarinov1994/baby-monitor/util"
	"gopkg.in/hraban/opus.v2"
)

const (
	sampleDurationMs  = 40
	opusFrameDuration = time.Millisecond * sampleDurationMs
	sampleRate        = 48000
	channels          = 2
	frameSize         = sampleRate * sampleDurationMs / 1000 // 40 ms at 48kHz

	scaleFactor = 3
)

var (
	chAudioRdy = make(chan struct{})
)

func startAudioFeed() {
	if err := portaudio.Initialize(); err != nil {
		util.CheckError(&err)
	}

	defer portaudio.Terminate()

	do, err := portaudio.DefaultInputDevice()
	if err != nil {
		util.CheckError(&err)
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
		util.CheckError(&err)
	}

	if err != istream.Start() {
		util.CheckError(&err)
	}

	defer istream.Close()

	encoder, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		util.CheckError(&err)
	}

	sampleData := util.CreateRingBuffer[[]int16](1)
	go func() {
		for data := range sampleData.Read() {
			if shutdown {
				return
			}

			sumSquares := 0.0
			for _, sample := range data {
				norm := float64(sample) / math.MaxInt16
				sumSquares += norm * norm
			}

			meanSquares := sumSquares / float64(len(data))
			rms := math.Sqrt(meanSquares)

			if rms < 1e-9 {
				dbPerFrame.Push(-96.0 * scaleFactor)
				log.Println("continue in channel range")
				continue
			}

			db := 20.0 * math.Log10(rms)
			if math.IsNaN(db) || math.IsInf(db, 0) {
				panic(fmt.Sprintf("Log10(%v) is NaN or Inf", rms))
			}

			dBapm := db * scaleFactor

			if dBapm > 0.0 {
				dBapm = 0.0
			}

			dbPerFrame.Push(dBapm)
		}

		log.Printf("Audio analyzer exited\n")
	}()

	close(chAudioRdy)
	for {
		if shutdown {
			return
		}

		// Read exactly one frame worth of PCM
		if err := istream.Read(); err != nil {
			util.CheckError(&err)
		}

		sampleData.Push(buffer)

		// Encode to Opus
		n, err := encoder.Encode(buffer, packet)
		if err != nil {
			util.CheckError(&err)
		}

		packet = packet[:n]
		audioFrames.Push(packet)
	}
}
