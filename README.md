# dgVoice : Add music playback support to Discordgo.

Include this along with Discordgo to add the ability to play audio files
to a server/channel.

You must have ffmpeg in your path and Opus libs already installed.

This is very incomplete and barely tested code.  Please understand that it
might not work and might have little glitches.  It does work on my system and
sounds pretty decent.

Please send feedback on any performance improvements that can be made for 
sound quality, stability, or efficiency.


# Usage Example
```
package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

func main() {

	var err error

	// Create a new Discord Session and set a handler for the OnMessageCreate
	// event that happens for every new message on any channel
	Session := discordgo.Session{
		OnMessageCreate: messageCreate,
	}

	// Login to the Discord server and store the authentication token
	// inside the Session
	Session.Token, err = Session.Login("username", "password")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Open websocket connection
	err = Session.Open()
	if err != nil {
		fmt.Println(err)
	}

	// Do websocket handshake.
	err = Session.Handshake()
	if err != nil {
		fmt.Println(err)
	}

	// Listen for events.
	go Session.Listen()

	// Connect to voice websocket.
	Session.VoiceChannelJoin("GuildIdNumberHere", "ChannelIdNumberHere")

	// Sleep is a very temp hack to give the voice channel enough
	// time to get connected.  I will add checks in later to see if we're
	// up and ready yet.
	time.Sleep(1 * time.Second)

	// streams file from ffmpeg then encodes with opus and sends via UDP
	// to Discord.
	dgvoice.PlayAudioFile(&Session, "mycoolsoundfile.flac")

	return
}

func messageCreate(s *discordgo.Session, m discordgo.Message) {
	fmt.Printf("%25d %s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)
}
```
