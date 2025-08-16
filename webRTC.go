package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

var (
	api *webrtc.API

	config = webrtc.Configuration{}

	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample
)

func createMediaEngine() {
	// Create a MediaEngine object to configure the supported codec
	mediaEngine := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
		},
		PayloadType: 105,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		checkError(&err)
	}

	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	// Create Track that we send video back to browser on
	vt, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "pion")
	if err != nil {
		checkError(&err)
	}

	at, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus,
	}, "audio", "pion")
	if err != nil {
		checkError(&err)
	}

	videoTrack = vt
	audioTrack = at
}

func handleConnection(res http.ResponseWriter, req *http.Request) {
	log.Println("CONNECT REQUEST")
	body, err := io.ReadAll(req.Body)
	if err != nil {
		checkError(&err)
	}

	clientOffer := webrtc.SessionDescription{}
	if err := json.Unmarshal(body, &clientOffer); err != nil {
		checkError(&err)
	}

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		checkError(&err)
	}

	// Add this newly created track to the PeerConnection
	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		checkError(&err)
	}

	// _, err = peerConnection.AddTrack(audioTrack)
	// if err != nil {
	// 	checkError(&err)
	// }

	var canditateJson webrtc.ICECandidateInit
	done := make(chan struct{})
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("OnICECandidate")
		if candidate != nil {
			canditateJson = candidate.ToJSON() // TODO: check formats or something
		} else {
			close(done)
		}
	})

	// Set the remote SessionDescription
	log.Printf("SetRemoteDescription")
	if err = peerConnection.SetRemoteDescription(clientOffer); err != nil {
		checkError(&err)
	}

	// Create answer
	log.Printf("CreateAnswer")
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		checkError(&err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	log.Printf("SetLocalDescription")
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		checkError(&err)
	}

	<-done

	log.Printf("Send anwser to server")
	response := map[string]any{
		"sdp":       answer.SDP,
		"type":      "answer",
		"candidate": canditateJson,
	}

	msg, err := json.Marshal(response)
	if err != nil {
		checkError(&err)
	}

	res.Header().Set("Content-Type", "application/json")
	send, err := res.Write(msg)
	if err != nil {
		checkError(&err)
	}

	if send != len(msg) {
		panic(fmt.Sprintf("Send (%d) != Response len (%d)", send, len(msg)))
	}
}

func fillVideoTrack(videoTrack *webrtc.TrackLocalStaticSample) {
	for {

		data := <-videoFrames.Read()
		if writeErr := videoTrack.WriteSample(
			media.Sample{
				Data:      data,
				Duration:  h264FrameDuration,
				Timestamp: time.Now(),
			}); writeErr != nil {
			checkError(&writeErr)
		}
	}
}

func fillAudioTrack(audioTrack *webrtc.TrackLocalStaticSample) {
	for {

		data := <-audioFrames.Read()
		if writeErr := audioTrack.WriteSample(
			media.Sample{
				Data:      data,
				Duration:  opusFrameDuration,
				Timestamp: time.Now(),
			}); writeErr != nil {
			checkError(&writeErr)
		}
	}
}
