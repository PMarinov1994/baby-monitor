package main

import (
	"log"
	"net/http"

	"githug.com/pmarinov1994/baby-monitor/mic"
)

var (
	soundCards []*mic.SoundCard

	videoFrames *ringBuffer[[]byte]
	audioFrames *ringBuffer[[]byte]
)

func main() {
	log.Printf("Enumerationg sound cards...\n")
	soundCards, err := mic.EnumSoundCards()
	if err != nil {
		log.Printf("Failed to enumerate sound card. Reason: %s\n", err)
		checkError(&err)
	}

	log.Printf("Found %d sound cards", len(soundCards))

	videoFrames = createRingBuffer[[]byte](1)
	audioFrames = createRingBuffer[[]byte](1)

	videoTrack, audioTrack = createMediaEngine()

	go startVideoFeed()
	go startAudioFeed()

	http.HandleFunc("/api", wsApiHandle)
	http.HandleFunc("/webRTCFeed", handleConnection)
	http.Handle("/", http.FileServer(http.Dir("./client/dist")))

	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatal(err)
	}
}
