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
	"githug.com/pmarinov1994/baby-monitor/mic"
)

type WsClient struct {
	ws *websocket.Conn
	id uuid.UUID
}

const (
	DATA_SEPARATOR = "&&&"

	REQ_SOUND_CARDS = "getSoundCards"
	RES_SOUND_CARDS = "gotSoundCards"

	REQ_CHANGE_SOUND = "setSound"
	RES_CHANGE_SOUND = "gotSound"
)

var (
	clients []*WsClient
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

	defer func() {
		log.Printf("[WebSocket] Cleaning client with id %s\n", client.id.String())
		client.ws.Close()
		for i, c := range clients {
			if c.id == client.id {
				clients[i] = nil
				break
			}
		}
	}()

	log.Printf("WebSocket client connected.\n")

	// Add the client to a free slot
	added := false
	for i := range clients {
		if clients[i] == nil {
			clients[i] = &client
			added = true
			break
		}
	}

	if !added {
		clients = append(clients, &client)
	}

	log.Printf("[WebSocket] Added client with id %s\n", client.id.String())

	for {
		t, msg, err := ws.ReadMessage()
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

		switch string(chunks[0]) {
		case REQ_SOUND_CARDS:
			processGetSoundCardsReq(ws)
		case REQ_CHANGE_SOUND:
			processVolumeChangeReq(chunks, ws)
		}
	}
}

func createSoundCardsRequest() []byte {
	jsonData, err := json.Marshal(soundCards)
	if err != nil {
		checkError(&err)
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
		checkError(&err)
	}
}

func processVolumeChangeReq(chunks []string, ws *websocket.Conn) {
	if len(chunks) < 4 {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(
			"%s%sError: invalid request for '%s'. Not enough data chunks",
			RES_CHANGE_SOUND,
			DATA_SEPARATOR,
			REQ_CHANGE_SOUND,
		)))
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
