package main

import "C"

import (
	"unsafe"

	"github.com/b1naryth1ef/telecom"
)

var clients map[*telecom.Client]bool = make(map[*telecom.Client]bool)

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
func telecom_client_destroy(client uintptr) bool {
	delete(clients, (*telecom.Client)(unsafe.Pointer(client)))
	return true
}

//export telecom_client_set_server_info
func telecom_client_set_server_info(clientPtr uintptr, endpoint, token *C.char) {
	client := (*telecom.Client)(unsafe.Pointer(clientPtr))
	client.SetServerInfo(C.GoString(endpoint), C.GoString(token))
}

//export telecom_client_play_from_file
func telecom_client_play_from_file(clientPtr uintptr, file *C.char) {
	client := (*telecom.Client)(unsafe.Pointer(clientPtr))
	filePath := C.GoString(file)

	go func() {
		client.WaitReady()
		telecom.PlayAudioFile(client, filePath)
	}()
}

func main() {}
