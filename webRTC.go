package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

var (
	mediaEngine webrtc.MediaEngine
	api         *webrtc.API
	config      webrtc.Configuration

	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample
)

func createMediaEngine() (*webrtc.TrackLocalStaticSample, *webrtc.TrackLocalStaticSample) {
	// Create a MediaEngine object to configure the supported codec
	mediaEngine = webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeH264,
			ClockRate:   90000,
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", // from mediamtx
		},
		PayloadType: 105,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		checkError(&err)
	}

	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1;stereo=1;sprop-stereo=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	// Prepare the configuration
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{},
	}

	// Create Track that we send video back to browser on
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "pion")
	if err != nil {
		checkError(&err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus,
	}, "audio", "pion")
	if err != nil {
		checkError(&err)
	}

	go fillVideoTrack(videoTrack)
	go fillAudioTrack(audioTrack)

	return videoTrack, audioTrack
}

func fillVideoTrack(videoTrack *webrtc.TrackLocalStaticSample) {
	for {

		data := <-videoFrames.Read()
		if writeErr := videoTrack.WriteSample(
			media.Sample{
				Data:     data,
				Duration: h264FrameDuration,
				//Timestamp: time.Now(),
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
				Data:     data,
				Duration: opusFrameDuration,
				//Timestamp: time.Now(),
			}); writeErr != nil {
			checkError(&writeErr)
		}
	}
}

func handleConnection(res http.ResponseWriter, req *http.Request) {
	log.Println("CONNECT REQUEST")
	body, _ := io.ReadAll(req.Body)

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
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		checkError(&err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		checkError(&err)
	}

	// Read incoming RTCP packets
	// Before these packets are retuned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	gatheringDone := make(chan struct{})
	newLocalCandidate := make(chan *webrtc.ICECandidateInit)

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())

		switch connectionState {
		case webrtc.ICEConnectionStateConnected:
			log.Println("ICEConnectionStateConnected")

		case webrtc.ICEConnectionStateFailed:
			log.Println("ICEConnectionStateFailed")

		case webrtc.ICEConnectionStateDisconnected:
			log.Println("ICEConnectionStateDisconnected")
		}
	})

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		if connectionState == webrtc.PeerConnectionStateConnected {
			log.Println("PeerConnectionStateConnected")
		}
	})

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		log.Printf("OnICECandidate")
		if i != nil {
			v := i.ToJSON()
			newLocalCandidate <- &v
		} else {
			close(gatheringDone)
		}
	})

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(clientOffer); err != nil {
		checkError(&err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		checkError(&err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		checkError(&err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	select {
	case <-newLocalCandidate:
	case <-gatheringDone:
	}

	msg, err := json.Marshal(*peerConnection.LocalDescription())
	if err != nil {
		checkError(&err)
	}

	send, err := res.Write(msg)
	if err != nil {
		checkError(&err)
	}

	if send != len(msg) {
		panic(fmt.Sprintf("Send (%d) != Response len (%d)", send, len(msg)))
	}
}
