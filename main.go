package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"githug.com/pmarinov1994/baby-monitor/mic"
	"githug.com/pmarinov1994/baby-monitor/util"
)

const (
	MAX_CONNECTED_CLIENT = 5
)

var (
	videoFrames *util.RingBuffer[[]byte] = util.CreateRingBuffer[[]byte](2)
	audioFrames *util.RingBuffer[[]byte] = util.CreateRingBuffer[[]byte](1)

	conClients uint8       = 0
	wsClients  []*WsClient = make([]*WsClient, MAX_CONNECTED_CLIENT)

	soundCards []*mic.SoundCard

	// TODO: Intercept interupt sig
	shutdown = false
)

func main() {
	log.Printf("GPIO Init...\n")
	initGpio()
	defer closeGpio()

	log.Printf("Enumerationg sound cards...\n")
	sc, err := mic.EnumSoundCards()
	if err != nil {
		log.Printf("Failed to enumerate sound card. Reason: %s\n", err)
		util.CheckError(&err)
	}

	soundCards = sc
	log.Printf("Found %d sound cards", len(soundCards))

	// NOTE: Create tracks before starting media,
	//       otherwize no video feed is present
	createMediaEngine()

	go startVideoFeed()
	go startAudioFeed()

	go sendAudioPkgs(audioTrack)
	go sendVideoPkgs(videoTrack)

	<-chAudioRdy
	log.Printf("Audio Ready!")
	<-chVideoRdy
	log.Printf("Video Ready!")

	http.HandleFunc("/api", wsApiHandle)
	http.HandleFunc("/webRTCFeed", handleConnection)
	http.Handle("/", http.FileServer(http.Dir("./client/dist")))

	ipAddr := getIpAddr()
	port := 8080

	log.Printf("======= Web Server Ready! =======\n")
	log.Printf("======= %s:%d =======\n", ipAddr, port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
		log.Fatal(err)
	}
}

func getIpAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		util.CheckError(&err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	panic("Failed to get IP Address")
}
