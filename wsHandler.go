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
	"githug.com/pmarinov1994/baby-monitor/util"
)

type WsClient struct {
	ws             *websocket.Conn
	id             uuid.UUID
	peerConnection *webrtc.PeerConnection
}

type WsClientState struct {
	NightVision bool `json:"nightVision"`
	DrawSound   bool `json:"drawSound"`
}

const (
	DATA_SEPARATOR = "&&&"

	// Client -> Server
	REQ_TOGGLE_NIGHT_VISION = "setToggleNightVision"
	RES_TOGGLE_NIGHT_VISION = "gotToggleNightVision"

	// Server -> Client
	REQ_UPDATE_STATE = "setUpdateState"
	RES_UPDATE_STATE = "gotUpdateState"

	// Client -> Server
	REQ_GET_STATE = "getGetState"
	RES_GET_STATE = "gotGetState"
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
		case REQ_TOGGLE_NIGHT_VISION:
			processToggleNightVisionReq(chunks)
			updateNeeded = true
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

func processToggleNightVisionReq(chunks []string) {
	toggle := chunks[1]

	b, err := strconv.ParseBool(toggle)
	if err != nil {
		util.CheckError(&err)
	}

	toggleNightVision(b)
	log.Printf("Setting night vision: %s\n", toggle)
}

func sendStateToClient(ws *websocket.Conn) {
	content := WsClientState{
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
