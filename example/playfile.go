package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

func main() {

	// NOTE: All of the below fields are required for this example to work correctly.
	var (
		Email     = flag.String("e", "", "Discord account email.")
		Password  = flag.String("p", "", "Discord account password.")
		GuildID   = flag.String("g", "", "Guild ID")
		ChannelID = flag.String("c", "", "Channel ID")
		Filename  = flag.String("f", "", "Filename of file to play.")
		err       error
	)
	flag.Parse()

	// Connect to Discord
	discord, err := discordgo.New(*Email, *Password)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err = discord.ChannelVoiceJoin(*GuildID, *ChannelID, false, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	// This will block until Voice is ready.  This is not the most ideal
	// way to check and shouldn't be used outside of this example.
	for {
		if discord.Voice.Ready {
			break
		}
		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}

	// streams file from ffmpeg then encodes with opus and sends via UDP
	// to Discord.
	dgvoice.PlayAudioFile(discord, *Filename)

	// Close connections
	discord.Voice.Close()
	discord.Close()

	return
}
