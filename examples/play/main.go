package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
		Folder    = flag.String("f", "", "Folder of files to play.")
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
	err = discord.ChannelVoiceJoin(*GuildID, *ChannelID, false, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	// This will block until Voice is ready.  This is not the most ideal
	// way to check and a better solution will be developed.
	// TODO : Improve this :)
	for {
		if discord.Voice.Ready {
			break
		}
		fmt.Print(".")
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("")

	// Start loop and attempt to play all files in the given folder
	fmt.Println("Reading Folder: %s", *Folder)
	files, _ := ioutil.ReadDir(*Folder)
	for _, f := range files {
		fmt.Println("PlayAudioFile:", f.Name())
		discord.UpdateStatus(0, f.Name())
		dgvoice.PlayAudioFile(discord, fmt.Sprintf("%s/%s", *Folder, f.Name()))
	}

	// Close connections
	discord.Voice.Close()
	discord.Close()

	return
}
