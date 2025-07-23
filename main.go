package main

import (
	"log"
	"net/http"
	"sync"

	"githug.com/pmarinov1994/baby-monitor/mic"
)

var soundCards []*mic.SoundCard
var mxClients sync.Mutex
var wsClients []*client

var mainAudioChannel *ringBuffer

func main() {
	log.Printf("Enumerationg sound cards...\n")
	soundCards, err := mic.EnumSoundCards()
	if err != nil {
		log.Printf("Failed to enumerate sound card. Reason: %s\n", err)
		panic(err)
	}

	log.Printf("Found %d sound cards", len(soundCards))

	mainAudioChannel = createRingBuffer(500)

	go process_sound()
	go transferSoundToClients()

	// TODO: Set max client count dinamically
	wsClients = make([]*client, 5)

	http.HandleFunc("/audio", wsAudioHandle)
	http.HandleFunc("/api", wsApiHandle)

	// WebPages handler
	http.Handle("/", http.FileServer(http.Dir("./front-end")))

	log.Printf("Starting server...")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
