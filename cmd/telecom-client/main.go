package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/b1naryth1ef/telecom"
)

var (
	userId    = flag.String("user-id", "", "user-id for the voice connection")
	guildId   = flag.String("guild-id", "", "guild-id for the voice connection")
	sessionId = flag.String("session-id", "", "session-id for the voice connection")
	token     = flag.String("token", "", "token for the voice connection")
	endpoint  = flag.String("endpoint", "", "initial endpoint")
)

func main() {
	flag.Parse()

	if userId == nil || guildId == nil || sessionId == nil || token == nil || endpoint == nil {
		fmt.Println("user-id, guild-id, session-id, endpoint, and token are required arguments")
		return
	}

	client := telecom.NewClient(*userId, *guildId, *sessionId)
	client.Run()

	if endpoint != nil && token != nil {
		client.UpdateServerInfo(*endpoint, *token)
	}

	<-client.Ready

	playable := telecom.NewAvConvPlayable("yeet.mp3")
	client.Play(playable)

	for {
		time.Sleep(1)
	}
}
