package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func wsApiHandle(writer http.ResponseWriter, request *http.Request) {
}

func wsAudioHandle(writer http.ResponseWriter, request *http.Request) {
	// FIXME: fix next line
	wsUpdater := websocket.Upgrader{}
	wsUpdater.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := wsUpdater.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("[WebSocket] Error %s when upgrading connection to websocket", err)
		return
	}

	client := client{
		audio_chain_ch: createRingBuffer(500),
	}

	id, err := addClient(wsClients, &client)
	if err != nil {
		log.Printf("[WebSocket][%d] New client rejected. Reason: %s\n", client.id, err)
		return
	}

	client.id = id

	log.Printf("[WebSocket][%d] Client connected\n", client.id)

	var wg sync.WaitGroup
	wg.Add(2)

	defer func() {
		log.Printf("[WebSocket][%d] Closing connection\n", client.id)
		ws.Close()
		removeClient(wsClients, &client)
		close(client.audio_chain_ch.ch)
	}()

	go func() {
		defer wg.Done()
		log.Printf("[WebSocket][%d] Starting read-loop (For client monitoring)", client.id)

		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				log.Printf("[WebSocket][%d] Failed to read from client. Reason: %s\n", client.id, err)
				break
			}
		}
	}()

	go func() {
		defer wg.Done()

		log.Printf("[WebSocket][%d] Starting write-loop (For sending data)", client.id)
		for frame := range client.audio_chain_ch.Read() {
			var packet bytes.Buffer
			err := binary.Write(&packet, binary.LittleEndian, frame)
			checkError(&err)

			ws.SetWriteDeadline(time.Now().Add(time.Second))
			err = ws.WriteMessage(websocket.BinaryMessage, packet.Bytes())
			if err != nil {
				log.Printf("[WebSocket][%d] Failed to send data to client. %s", client.id, err)
				return
			}
		}
	}()

	wg.Wait()
}
