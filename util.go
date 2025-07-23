package main

import "errors"

func checkError(err *error) {
	if *err != nil {
		panic(err)
	}
}

func addClient(list []*client, client *client) (int, error) {
	mxClients.Lock()
	defer mxClients.Unlock()

	for i, c := range list {
		if c == nil {
			list[i] = client
			return i, nil
		}
	}

	return -1, errors.New("No client slots available. Server is full.")
}

func removeClient(list []*client, client *client) {
	mxClients.Lock()
	defer mxClients.Unlock()

	list[client.id] = nil
}

func transferSoundToClients() {
	for frame := range mainAudioChannel.Read() {
		// NOTE: Wrap inside a function so that defer will
		// trigger after all clients are looped
		// If not wrapped, defer will be called only when the 'for channel' closes
		func() {
			mxClients.Lock()
			defer mxClients.Unlock()

			for _, c := range wsClients {
				if c != nil {
					buf := make([]float32, len(frame))
					copy(buf, frame)
					c.audio_chain_ch.Push(buf)
				}
			}
		}()
	}
}
