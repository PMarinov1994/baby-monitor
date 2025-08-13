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
	sc, err := mic.EnumSoundCards()
	if err != nil {
		log.Printf("Failed to enumerate sound card. Reason: %s\n", err)
		checkError(&err)
	}

	soundCards = sc
	log.Printf("Found %d sound cards", len(soundCards))

	videoFrames = createRingBuffer[[]byte](1)
	audioFrames = createRingBuffer[[]byte](1)

	// NOTE: Create tracks before starting media,
	//       otherwize no video feed is present
	videoTrack, audioTrack = createMediaEngine()

	if isVideoSourceAvailable() {
		log.Printf("Starting video feed.\n")
		go startVideoFeed()
	} else {
		log.Printf("No video feed. Not running on Raspberry Pi.\n")
	}

	go startAudioFeed()

	http.HandleFunc("/api", wsApiHandle)
	http.HandleFunc("/webRTCFeed", handleConnection)
	http.Handle("/", http.FileServer(http.Dir("./client/dist")))

	log.Printf("Web Server Ready!\n")
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatal(err)
	}
}
