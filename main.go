package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"githug.com/pmarinov1994/baby-monitor/util"
)

const (
	MAX_CONNECTED_CLIENT     = 5
	VIDEO_FRAME_BUFFER_COUNT = 2
)

var (
	conClients uint8       = 0
	wsClients  []*WsClient = make([]*WsClient, MAX_CONNECTED_CLIENT)

	// TODO: Intercept interupt sig
	shutdown = false
)

func main() {
	log.Printf("GPIO Init...\n")
	initGpio()
	defer closeGpio()

	http.HandleFunc("/api", wsApiHandle)
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
