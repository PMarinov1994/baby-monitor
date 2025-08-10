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

	"github.com/mattetti/audio"
	// "github.com/mattetti/audio/transforms/filters"
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
		"-f", "FLOAT_LE", //"S16_LE",
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
	pcm := make([]float64, frameSize*channels) // float32 PCM samples
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
			pcm[i] = float64(math.Float32frombits(bits))
		}

		// pcmLowPass := lowPassFilter(pcm, 3000, outSampleRate)
		// pcm64 := float32ToFloat64(pcm)
		aBuffer := audio.NewPCMFloatBuffer(pcm, &audio.Format{
			SampleRate: outSampleRate,
		})

		// err := filters.HighPass(aBuffer, 200)
		// if err != nil {
		// 	checkError(&err)
		// }

		// Encode to Opus
		n, err := encoder.EncodeFloat32(aBuffer.AsFloat32s(), packet)
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

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, f64 := range in {
		out[i] = float32(f64)
	}

	return out
}

func lowPassFilter(samples []float32, cutoff, sampleRate float32) []float32 {
	rc := 1.0 / (2 * math.Pi * cutoff)
	dt := 1.0 / sampleRate
	alpha := dt / (rc + dt)

	output := make([]float32, len(samples))
	if len(samples) == 0 {
		return output
	}

	output[0] = samples[0]
	for i := 1; i < len(samples); i++ {
		output[i] = output[i-1] + alpha*(samples[i]-output[i-1])
	}

	return output
}
