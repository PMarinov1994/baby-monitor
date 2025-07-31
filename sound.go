package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	// "atomicgo.dev/cursor"
	"github.com/gordonklaus/portaudio"
)

const sampleRate = 48000
const frameBuffer = 1024

func process_sound() {
	log.Println("Starting sound driver...")

	err := portaudio.Initialize()
	checkError(&err)

	defer portaudio.Terminate()

	devices, err := portaudio.Devices()
	checkError(&err)

	di, err := portaudio.DefaultInputDevice()
	checkError(&err)

	do, err := portaudio.DefaultOutputDevice()
	checkError(&err)

	var rpi_code_zero *portaudio.DeviceInfo

	max_name_len := 0
	max_samplerate_len := 0
	max_latency_len := 0
	for _, d := range devices {
		max_name_len = max(max_name_len, len(d.Name))
		max_samplerate_len = max(max_samplerate_len, len(fmt.Sprintf("%f", d.DefaultSampleRate)))
		max_latency_len = max(max_latency_len, len(d.DefaultLowInputLatency.String()))
	}

	fmt.Printf("\n\n D | Name | SampleRate | Latency | In channels | Out channels\n")
	fmt.Printf("----------------------------------------------\n")
	for _, d := range devices {
		mark := "   "
		if do == di && di == d {
			mark = " <>"
		} else if di == d {
			mark = "  >"
		} else if do == d {
			mark = "  <"
		}

		if strings.Contains(d.Name, "Codec Zero") {
			rpi_code_zero = d
		}

		fmt.Printf("%s%*s | %*f | %*s | %2d | %2d\n",
			mark,
			max_name_len*-1, d.Name,
			max_samplerate_len, d.DefaultSampleRate,
			max_latency_len, d.DefaultLowInputLatency.String(),
			d.MaxInputChannels,
			d.MaxOutputChannels)
	}
	fmt.Printf("----------------------------------------------\n")

	// Input
	in_stream_channels := 1
	in_params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   rpi_code_zero,
			Channels: in_stream_channels,
			Latency:  rpi_code_zero.DefaultLowInputLatency,
		},
		SampleRate:      sampleRate,
		FramesPerBuffer: frameBuffer * in_stream_channels,
	}

	in_stream, err := portaudio.OpenStream(in_params, func(in []float32) {
		buf := make([]float32, len(in))
		copy(buf, in)
		mainAudioChannel.Push(buf)
	})

	checkError(&err)

	err = in_stream.Start()
	checkError(&err)

	defer in_stream.Close()

	if rpi_code_zero == nil {
		panic("Rpi-codec-zero dev output is nil")
	}

	// Output
	// out_params := portaudio.StreamParameters{
	// 	Output: portaudio.StreamDeviceParameters{
	// 		Device:   hdmi_dev,
	// 		Channels: hdmi_dev.MaxOutputChannels,
	// 		Latency:  hdmi_dev.DefaultLowOutputLatency,
	// 	},
	// 	SampleRate:      sampleRate,
	// 	FramesPerBuffer: frameBuffer * hdmi_dev.MaxOutputChannels * in_stream_channels,
	// }

	// out_stream, err := portaudio.OpenStream(out_params, func(out []float32) {
	// 	for i, j := 0, 0; i < len(out); i, j = i+2, j+1 {
	// 		out[i] = buffer[j]
	// 		out[i+1] = buffer[j]
	// 	}
	// })
	//
	// check_error(&err)
	//
	// err = out_stream.Start()
	// check_error(&err)
	//
	// defer out_stream.Close()

	for {
		time.Sleep(time.Millisecond)
	}
}
