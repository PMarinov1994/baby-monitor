package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"

	"githug.com/pmarinov1994/baby-monitor/util"
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

	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample

	audioPkgs *util.RingBuffer[*media.Sample] = util.CreateRingBuffer[*media.Sample](1)
	videoPkgs *util.RingBuffer[*media.Sample] = util.CreateRingBuffer[*media.Sample](1)
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

	// Create a InterceptorRegistry. This is the user configurable RTP/RTCP Pipeline.
	// This provides NACKs, RTCP Reports and other features. If you use `webrtc.NewPeerConnection`
	// this is enabled by default. If you are manually managing You MUST create a InterceptorRegistry
	// for each PeerConnection.
	interceptorRegistry := &interceptor.Registry{}

	// Use the default set of Interceptors
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		checkError(&err)
	}

	// Create the API object with the MediaEngine
	api = webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry))

	// Create Track that we send video back to browser on
	vt, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeH264,
		ClockRate: h264ClockRate,
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

	if conClients >= MAX_CONNECTED_CLIENT {
		log.Printf("Rejected new client connection")
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
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		checkError(&err)
	}

	if _, err := peerConnection.AddTrack(audioTrack); err != nil {
		checkError(&err)
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	processRTCP := func(rtpSender *webrtc.RTPSender, wg *sync.WaitGroup) {
		defer wg.Done()

		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				log.Printf("Process RTCP done %v\n", rtcpErr)
				return
			}
		}
	}

	var senderDisconnWaitGr sync.WaitGroup
	for _, rtpSender := range peerConnection.GetSenders() {
		senderDisconnWaitGr.Add(1)
		go processRTCP(rtpSender, &senderDisconnWaitGr)
	}

	var canditates []webrtc.ICECandidateInit
	iceGatherDone := make(chan struct{})
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			canditates = append(canditates, candidate.ToJSON()) // TODO: check formats or something
		} else {
			log.Println("Gathering Done!")
			close(iceGatherDone)
		}
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateClosed:
			log.Println("Peer Connection has gone to closed")
		case webrtc.PeerConnectionStateConnected:
			log.Println("Peer Connection connected")
		}
	})

	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", dataChannel.Label(), dataChannel.ID())

		dataChannel.OnOpen(func() {
			log.Printf("DataChannel Opened %s %d\n", dataChannel.Label(), *dataChannel.ID())
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			strMsg := string(msg.Data)
			log.Printf("Got message from DataChannel %s\n", strMsg)

			found := false
			for _, c := range wsClients {
				if c.id.String() == strMsg {
					conClients++
					log.Printf("ConnClients: %d\n", conClients)
					c.peerConnection = peerConnection
					found = true
					break
				}
			}

			if !found {
				log.Printf("Could not find client with id %s\n", strMsg)
			}

			dataChannel.Send(fmt.Appendf(nil, "OK"))
		})

		dataChannel.OnClose(func() {
			log.Printf("DataChannel Closed %s %d\n", dataChannel.Label(), *dataChannel.ID())
		})
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

	log.Println("Waiting for gathering to complete...")
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

func sendAudioPkgs(audioTrack *webrtc.TrackLocalStaticSample) {
	for pkg := range audioPkgs.Read() {
		if writeErr := audioTrack.WriteSample(*pkg); writeErr != nil {
			checkError(&writeErr)
		}
	}
}

func sendVideoPkgs(videoTrack *webrtc.TrackLocalStaticSample) {
	for pkg := range videoPkgs.Read() {
		if writeErr := videoTrack.WriteSample(*pkg); writeErr != nil {
			checkError(&writeErr)
		}
	}
}

func createPkgs() {
	for {

		audioData := <-audioFrames.Read()
		videoData := <-videoFrames.Read()

		now := time.Now()

		audioPkg := &media.Sample{
			Data:      audioData,
			Duration:  opusFrameDuration,
			Timestamp: now,
		}

		videoPkg := &media.Sample{
			Data:      videoData,
			Duration:  h264FrameDuration,
			Timestamp: now,
		}

		audioPkgs.Push(audioPkg)
		videoPkgs.Push(videoPkg)
	}
}
