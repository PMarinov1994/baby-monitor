package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
)

const (
	h264PayloadType = 105
	h264ClockRate   = 90000

	opusPayloadType = 111
	opusClockRate   = 48000
)

var (
	api *webrtc.API

	config = webrtc.Configuration{}

	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
)

func createMediaEngine() {
	// Create a MediaEngine object to configure the supported codec
	mediaEngine := &webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: h264ClockRate,
		},
		PayloadType: h264PayloadType,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		checkError(&err)
	}

	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: opusClockRate,
			Channels:  2,
		},
		PayloadType: opusPayloadType,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		checkError(&err)
	}

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry))

	// Create Track that we send video back to browser on
	vt, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeH264,
		ClockRate: h264ClockRate,
	}, "video", "pion")
	if err != nil {
		checkError(&err)
	}

	at, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
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

	if connectedClients >= MAX_CONNECTED_CLIENT {
		http.Error(res, "No connection spots left", 500)
		return
	}

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
	rtpVideoSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		checkError(&err)
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 32)
		for {
			if _, _, rtcpErr := rtpVideoSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	rtpAudioSender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		checkError(&err)
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 32)
		for {
			if _, _, rtcpErr := rtpAudioSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	var canditates []webrtc.ICECandidateInit
	iceGatherDone := make(chan struct{})
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			canditates = append(canditates, candidate.ToJSON()) // TODO: check formats or something
		} else {
			close(iceGatherDone)
		}
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateClosed:
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			connectedClients--
			log.Printf("connectedClients val: %d\n", connectedClients)
			log.Println("Peer Connection has gone to closed. Closing connection.")
			if err := peerConnection.Close(); err != nil {
				checkError(&err)
			}
		case webrtc.PeerConnectionStateConnected:
			connectedClients++
			log.Printf("connectedClients val: %d\n", connectedClients)
			log.Println("Peer Connection connected")
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
	// This will trigger the ICE Candidate gathering
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		checkError(&err)
	}

	<-iceGatherDone

	response := map[string]any{
		"sdp":        answer.SDP,
		"type":       "answer",
		"candidates": canditates,
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

func fillVideoTrack(videoTrack *webrtc.TrackLocalStaticRTP) {
	// Create a packetizer for H.264
	packetizer := rtp.NewPacketizer(
		1200,                     // MTU
		h264PayloadType,          // Payload type (dynamic, adjust as needed)
		12345,                    // SSRC
		&codecs.H264Payloader{},  // Payloader for H.264
		rtp.NewFixedSequencer(0), // Sequencer for RTP sequence numbers
		h264ClockRate,            // Clock rate for H.264
	)

	for {

		data := <-videoFrames.Read()
		rtpPackets := packetizer.Packetize(data, 3600)
		for _, packet := range rtpPackets {
			if err := videoTrack.WriteRTP(packet); err != nil {
				checkError(&err)
			}
		}
	}
}

func fillAudioTrack(audioTrack *webrtc.TrackLocalStaticRTP) {
	// Create a packetizer for Opus
	packetizer := rtp.NewPacketizer(
		1200,                     // MTU
		opusPayloadType,          // Payload type (dynamic, adjust as needed)
		54321,                    // SSRC
		&codecs.OpusPayloader{},  // Payloader for Opus
		rtp.NewFixedSequencer(0), // Sequencer for RTP sequence numbers
		opusClockRate,            // Clock rate for Opus
	)

	for {

		data := <-audioFrames.Read()
		rtpPackets := packetizer.Packetize(data, 960)
		for _, packet := range rtpPackets {
			if err := audioTrack.WriteRTP(packet); err != nil {
				checkError(&err)
			}
		}
	}
}
