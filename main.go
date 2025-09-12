package main

import (
	"log"
	"net/http"

	"githug.com/pmarinov1994/baby-monitor/mic"
)

const (
	MAX_CONNECTED_CLIENT = 5
)

var (
	videoFrames *ringBuffer[[]byte] = createRingBuffer[[]byte](4)
	audioFrames *ringBuffer[[]byte] = createRingBuffer[[]byte](1)

	conClients uint8       = 0
	wsClients  []*WsClient = make([]*WsClient, MAX_CONNECTED_CLIENT)

	soundCards []*mic.SoundCard
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

	// NOTE: Create tracks before starting media,
	//       otherwize no video feed is present
	createMediaEngine()

	if isVideoSourceAvailable() {
		log.Printf("Starting video feed.\n")
		go startVideoFeed()
	} else {
		log.Printf("No video feed. Not running on Raspberry Pi.\n")
	}

	go startAudioFeed()

	<-chAudioRdy
	log.Printf("Audio Ready!")
	<-chVideoRdy
	log.Printf("Video Ready!")

	go createPkgs()
	go sendAudioPkgs(audioTrack)
	go sendVideoPkgs(videoTrack)

	http.HandleFunc("/api", wsApiHandle)
	http.HandleFunc("/webRTCFeed", handleConnection)
	http.Handle("/", http.FileServer(http.Dir("./client/dist")))

	log.Printf("======= Web Server Ready! =======\n")
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatal(err)
	}
}
