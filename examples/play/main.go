package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"

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

	// Hacky loop to prevent sending on a nil channel.
	// TODO: Find a better way.
	for discord.Voice.Ready == false {
		runtime.Gosched()
	}

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
