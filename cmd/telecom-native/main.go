package main

import "C"

import (
	"os"
	"unsafe"

	"github.com/b1naryth1ef/telecom"
	log "github.com/sirupsen/logrus"
)

type PlayableHandle struct {
	P telecom.Playable
}

var clients map[*telecom.Client]bool = make(map[*telecom.Client]bool)
var playables map[*PlayableHandle]bool = make(map[*PlayableHandle]bool)

//export telecom_setup_logging
func telecom_setup_logging(enable bool, debug bool) {
	if !enable {
		log.SetLevel(log.ErrorLevel)
	} else if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetOutput(os.Stderr)
}

//export telecom_create_client
func telecom_create_client(userId, guildId, sessionId *C.char) uintptr {
	client := telecom.NewClient(
		C.GoString(userId),
		C.GoString(guildId),
		C.GoString(sessionId),
	)
	clients[client] = true
	client.Run()
	return uintptr(unsafe.Pointer(client))
}

//export telecom_client_destroy
func telecom_client_destroy(clientPtr uintptr) {
	client := (*telecom.Client)(unsafe.Pointer(clientPtr))
	// TODO: rename `disconnect`
	client.Disconnect()
	delete(clients, client)
}

//export telecom_client_update_server_info
func telecom_client_update_server_info(clientPtr uintptr, endpoint, token *C.char) {
	client := (*telecom.Client)(unsafe.Pointer(clientPtr))
	client.UpdateServerInfo(C.GoString(endpoint), C.GoString(token))
}

//export telecom_client_play
func telecom_client_play(clientPtr, playablePtr uintptr) {
	client := (*telecom.Client)(unsafe.Pointer(clientPtr))
	playable := (*PlayableHandle)(unsafe.Pointer(playablePtr))
	go func() {
		client.Play(playable.P)
	}()
}

//export telecom_create_avconv_playable
func telecom_create_avconv_playable(source *C.char) uintptr {
	playable := telecom.NewAvConvPlayable(C.GoString(source))
	handle := &PlayableHandle{playable}
	playables[handle] = true
	return uintptr(unsafe.Pointer(handle))
}

//export telecom_playable_destroy
func telecom_playable_destroy(playablePtr uintptr) {
	playable := (*PlayableHandle)(unsafe.Pointer(playablePtr))
	delete(playables, playable)
}

func main() {}
