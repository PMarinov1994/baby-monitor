package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"githug.com/pmarinov1994/baby-monitor/mic"
	"githug.com/pmarinov1994/baby-monitor/util"
)

type WsClient struct {
	ws             *websocket.Conn
	id             uuid.UUID
	peerConnection *webrtc.PeerConnection
}

type WsClientState struct {
	SoundCards  []*mic.SoundCard `json:"soundCards"`
	NightVision bool             `json:"nightVision"`
	DrawSound   bool             `json:"drawSound"`
}

const (
	DATA_SEPARATOR = "&&&"

	// Client -> Server
	REQ_SOUND_CARDS = "getSoundCards"
	RES_SOUND_CARDS = "gotSoundCards"

	// Client -> Server
	REQ_CHANGE_SOUND = "setSound"
	RES_CHANGE_SOUND = "gotSound"

	// Client -> Server
	REQ_WS_ID = "getWsId"
	RES_WS_ID = "setWsId"

	// Client -> Server
	REQ_TOGGLE_NIGHT_VISION = "setToggleNightVision"
	RES_TOGGLE_NIGHT_VISION = "gotToggleNightVision"

	// Client -> Server
	REQ_TOGGLE_SOUND_DRAW = "setToggleSoundDraw"
	RES_TOGGLE_SOUND_DRAW = "gotToggleSoundDraw"

	// Server -> Client
	REQ_UPDATE_STATE = "setUpdateState"
	RES_UPDATE_STATE = "gotUpdateState"

	// Client -> Server
	REQ_GET_STATE = "getGetState"
	RES_GET_STATE = "gotGetState"

	// Client -> Server
	REQ_SET_ROTATE = "getSetRotate"
	RES_SET_ROTATE = "gotSetRotate"
)

func wsApiHandle(writer http.ResponseWriter, request *http.Request) {
	wsUpdater := websocket.Upgrader{}

	// FIXME: We need to check origin or something
	wsUpdater.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := wsUpdater.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("[WebSocket] Error %s when upgrading connection to websocket", err)
		return
	}

	client := WsClient{
		ws: ws,
		id: uuid.New(),
	}

	log.Printf("WebSocket client connected.\n")

	if conClients >= MAX_CONNECTED_CLIENT {
		if err := ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil, "No client spots left")); err != nil {
			util.CheckError(&err)
		}
		log.Printf("Rejecting WebSocket Upgrade\n")
		return
	}

	// Add the client to a free slot
	for i := range wsClients {
		if wsClients[i] == nil {
			wsClients[i] = &client
			break
		}
	}

	log.Printf("[WebSocket] Added client with id %s\n", client.id.String())

	go handleWsClient(client)
}

func handleWsClient(client WsClient) {
	defer func() {
		log.Printf("[WebSocket] Cleaning client with id %s\n", client.id.String())
		for i, c := range wsClients {
			if c != nil && c.id == client.id {
				wsClients[i] = nil
				break
			}
		}

		if err := client.ws.Close(); err != nil {
			util.CheckError(&err)
		}

		if client.peerConnection != nil {
			if err := client.peerConnection.Close(); err != nil {
				// Ignore error on close
				util.CheckError(&err)
			}
		}

		conClients--
		log.Printf("ConnClients: %d\n", conClients)

		if conClients == 0 {
			toggleGreenLED(false)
		}
	}()

	for {
		if shutdown {
			return
		}

		t, msg, err := client.ws.ReadMessage()
		if err != nil {
			log.Printf("[WebSocket] Client (%s) error: %v\n", client.id.String(), err)
			break
		}

		if t != websocket.TextMessage {
			log.Printf("[WebSocket] Only text based messages are allowed (%s)\n", client.id.String())
			break
		}

		log.Printf("[WebSocket] Got message from %s: %s", client.id.String(), msg)

		req := string(msg)
		chunks := strings.Split(req, DATA_SEPARATOR)

		updateNeeded := false
		switch string(chunks[0]) {
		case REQ_SOUND_CARDS:
			processGetSoundCardsReq(client.ws)

		case REQ_CHANGE_SOUND:
			processVolumeChangeReq(chunks, client.ws)
			updateNeeded = true

		case REQ_WS_ID:
			processWsIdReq(client.id, client.ws)

		case REQ_TOGGLE_NIGHT_VISION:
			processToggleNightVisionReq(chunks)
			updateNeeded = true

		case REQ_TOGGLE_SOUND_DRAW:
			processToggleSoundDraw(chunks)
			updateNeeded = true

		case REQ_GET_STATE:
			sendStateToClient(client.ws)

		case REQ_SET_ROTATE:
			processSetRotate(chunks, client.ws)
		}

		if updateNeeded {
			for _, currWs := range wsClients {
				if currWs != nil && currWs.id.String() != client.id.String() {
					sendStateToClient(currWs.ws)
				}
			}
		}
	}
}

func createSoundCardsRequest() []byte {
	jsonData, err := json.Marshal(soundCards)
	if err != nil {
		util.CheckError(&err)
	}

	resHeaderLen := len(RES_SOUND_CARDS)
	resSeparatorLen := len(DATA_SEPARATOR)

	response := make([]byte, resHeaderLen+resSeparatorLen+len(jsonData))

	copy(response, []byte(RES_SOUND_CARDS))
	copy(response[resHeaderLen:], []byte(DATA_SEPARATOR))
	copy(response[resHeaderLen+resSeparatorLen:], jsonData)

	return response
}

func processGetSoundCardsReq(ws *websocket.Conn) {
	response := createSoundCardsRequest()
	if err := ws.WriteMessage(websocket.TextMessage, response); err != nil {
		util.CheckError(&err)
	}
}

func processVolumeChangeReq(chunks []string, ws *websocket.Conn) {
	if len(chunks) < 4 {
		ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
			"%s%sError: invalid request for '%s'. Not enough data chunks",
			RES_CHANGE_SOUND,
			DATA_SEPARATOR,
			REQ_CHANGE_SOUND,
		))
		return
	}

	var outputCh *mic.OutputChannel

	soundCardName := chunks[1]
	outputChName := chunks[2]
	newValueStr := chunks[3]

	for _, sd := range soundCards {
		if sd.LongName == soundCardName {
			for _, ch := range sd.OutputChannels {
				if ch.Name == outputChName {
					outputCh = ch
					break
				}
			}
		}
	}

	if outputCh == nil {
		ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
			"%s%sError: invalid request for '%s'. Channel (%s) from %s not found",
			RES_CHANGE_SOUND,
			DATA_SEPARATOR,
			REQ_CHANGE_SOUND,
			outputChName,
			soundCardName,
		))
		return
	}

	newValue, err := strconv.Atoi(newValueStr)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
			"%s%sError: invalid request for '%s'. New value (%s) not an integer",
			RES_CHANGE_SOUND,
			DATA_SEPARATOR,
			REQ_CHANGE_SOUND,
			newValueStr,
		))
		return
	}

	res, err := outputCh.SetVolume(newValue)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
			"%s%sError: Failed to set volume. Reason %v",
			RES_CHANGE_SOUND,
			DATA_SEPARATOR,
			err,
		))
	}

	log.Printf("Change volume result: %t\n", res)
}

func processWsIdReq(id uuid.UUID, ws *websocket.Conn) {
	ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
		"%s%s%s",
		RES_WS_ID,
		DATA_SEPARATOR,
		id.String(),
	))

	log.Printf("Sending WsClient ID: %s\n", id.String())
}

func processToggleNightVisionReq(chunks []string) {
	toggle := chunks[1]

	b, err := strconv.ParseBool(toggle)
	if err != nil {
		util.CheckError(&err)
	}

	toggleNightVision(b)
	log.Printf("Setting night vision: %s\n", toggle)
}

func processToggleSoundDraw(chunks []string) {
	toggle := chunks[1]

	b, err := strconv.ParseBool(toggle)
	if err != nil {
		util.CheckError(&err)
	}

	enableSoundDraw = b
	log.Printf("Setting sound draw: %s\n", toggle)
}

func sendStateToClient(ws *websocket.Conn) {
	content := WsClientState{
		SoundCards:  soundCards,
		DrawSound:   enableSoundDraw,
		NightVision: isNightVisionOn,
	}

	jsonData, err := json.Marshal(content)
	if err != nil {
		util.CheckError(&err)
	}

	resHeaderLen := len(REQ_UPDATE_STATE)
	resSeparatorLen := len(DATA_SEPARATOR)

	request := make([]byte, resHeaderLen+resSeparatorLen+len(jsonData))

	copy(request, []byte(REQ_UPDATE_STATE))
	copy(request[resHeaderLen:], []byte(DATA_SEPARATOR))
	copy(request[resHeaderLen+resSeparatorLen:], jsonData)

	if err := ws.WriteMessage(websocket.TextMessage, request); err != nil {
		util.CheckError(&err)
	}
}

func processSetRotate(chunks []string, ws *websocket.Conn) {
	if len(chunks) != 2 &&
		chunks[1] != "-90" && chunks[1] != "90" {
		ws.WriteMessage(websocket.TextMessage, fmt.Appendf(nil,
			"%s%sError: Invalid parameters",
			RES_SET_ROTATE,
			DATA_SEPARATOR,
		))
	}

	r, _ := strconv.Atoi(chunks[1]) // We checked the string above

	newRotation := int(vidFrameRotation)
	newRotation += r

	if newRotation > 270 || newRotation < 0 || newRotation%90 != 0 {
		newRotation = 0
	}

	vidFrameRotation = Rotation(newRotation)
}
