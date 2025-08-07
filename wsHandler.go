package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
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

	defer ws.Close()
}
