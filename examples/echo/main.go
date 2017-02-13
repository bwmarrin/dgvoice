package main

import (
	"flag"
	"fmt"

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
		err       error
	)
	flag.Parse()

	// Connect to Discord
	discord, err := discordgo.New(*Email, *Password)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Open Websocket
	err = discord.Open()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	dgv, err := discord.ChannelVoiceJoin(*GuildID, *ChannelID, false, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Starts echo
	echo(dgv)

	// Close connections
	dgv.Close()
	discord.Close()

	return
}

// Takes inbound audio and sends it right back out.
func echo(v *discordgo.VoiceConnection) {

	recv := make(chan *discordgo.Packet, 2)
	go dgvoice.ReceivePCM(v, recv)

	send := make(chan []int16, 2)
	go dgvoice.SendPCM(v, send)

	v.Speaking(true)
	defer v.Speaking(false)

	for {

		p, ok := <-recv
		if !ok {
			return
		}

		send <- p.PCM
	}
}
