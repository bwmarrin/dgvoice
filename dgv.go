package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	Discord "github.com/bwmarrin/discordgo"
	Opus "github.com/layeh/gopus"
)

var (
	Session  Discord.Session
	Username string
	Password string
	Guild    Discord.Guild   // active Guild
	Channel  Discord.Channel // active Channel
	err      error
)

func playAudioFile(filename string) {

	// assigns a NewEncoder to oe
	oe, err := Opus.NewEncoder(48000, 1, Opus.Audio)
	chk(err)
	oe.SetVbr(true)
	//oe.SetBitrate(Opus.BitrateMaximum)

	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", "sample.wav", "-f", "s16le", "-ar", "48000", "-ac", "1", "pipe:1")
	stdout, err := run.StdoutPipe()
	chk(err)

	// Starts the ffmpeg command
	err = run.Start()
	chk(err)

	// variables used during loop below
	udpPacket := make([]byte, 4000) // probably way bigger than needed :)
	audiobuf := make([]int16, 960)
	var sequence uint16 = 0
	var timestamp uint32 = 0

	// build the parts that don't change in the udpPacket.
	udpPacket[0] = 0x80
	udpPacket[1] = 0x78
	binary.BigEndian.PutUint32(udpPacket[8:], Session.Vop2.SSRC)

	// Send "speaking" packet over the voice websocket
	Session.VoiceSpeaking()

	// start a read/encode/send loop that loops until EOF from ffmpeg
	for {

		if sequence > 100 {
			return // just bail out to avoid too much spam.
		}

		sequence += 1    // this just increments each loop
		timestamp += 960 // also just increments each loop
		// TODO add a check so they don't go over value

		// Add sequence and timestamp to udpPacket
		binary.BigEndian.PutUint16(udpPacket[2:], sequence)
		binary.BigEndian.PutUint32(udpPacket[4:], timestamp)

		// read 1920 bytes (960 int16) from ffmpeg stdout
		err = binary.Read(stdout, binary.LittleEndian, &audiobuf)

		if err == io.EOF {
			fmt.Println("Reached EOF.")
			return
		}
		chk(err)

		// try encoding ffmpeg frame with Opus
		opus, err := oe.Encode(audiobuf, 960, 1920)
		chk(err)
		fmt.Println("Got ", len(opus), " byte array from Opus")

		// copy opus result into udpPacket
		z := copy(udpPacket[12:], opus)
		fmt.Println("Copied", z, " bytes from Opus to udpPacket")

		// write the udp packet to the active udp connection
		fmt.Println("Send UDP: SEQ:", sequence, " TS:", timestamp, "SIZE:", len(opus)+12)
		Session.UDPConn.Write(udpPacket[:(len(opus) + 12)])

		time.Sleep(20 * time.Millisecond) // sleep for 20ms
	}

	fmt.Println("playAudioFile finished")
}

func main() {

	Session = Discord.Session{
		OnEvent: OnEvent,
		OnReady: OnReady,
	}

	fmt.Printf("\nDiscordgo Console :  Type ~help for help. \n\n")

	// Reads in commands from file just as if they were typed at console.
	readFile()

	// Listens to stdin for console commands.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		line := scanner.Text()

		if strings.HasPrefix(line, "~") {
			response := parse(line)
			fmt.Println(response)
			continue
		}
	}
}

// OnEvent is called for unknown events or unhandled events.  It provides
// a generic interface to handle them.
func OnEvent(s *Discord.Session, e Discord.Event) {
	fmt.Println("Got Event: ", e.Type)
}

// OnReady is called when Discordgo receives a READY event
// This event must be handled and must contain the Heartbeat call.
func OnReady(s *Discord.Session, st Discord.Ready) {

	s.SessionID = st.SessionID

	// start the Heartbeat
	go s.Heartbeat(st.HeartbeatInterval)

	// Add code here to handle this event.
}

// See below for example file contents.
/*
username USERNAME_HERE
password PASSWORD_HERE
login
guild GUILD_ID_HERE
channel CHANNEL_ID_HERE
listen
debug
vconnect
*/
// once all of those commands finish, try the type ~vplay into the
// console.
//
// Read in commands from .dgcrc file and runs them one by one.
func readFile() {
	file, err := os.Open(".dgcrc")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		fmt.Println(parse(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// parses all config or console commands
func parse(line string) (response string) {

	// split the command from the rest
	split := strings.SplitN(line, " ", 2)

	// store the command and command options seperately
	cmd := strings.ToLower(split[0])
	cmd = strings.TrimPrefix(cmd, "~")

	var cmdOpts string = ""

	if len(split) > 1 {
		cmdOpts = split[1]
	}

	if cmd == "help" {
		response += "\n"
		response += fmt.Sprintln("~help ____________ Display this Help text.")
		response += fmt.Sprintln("~username ________ Set the Username for ~login.")
		response += fmt.Sprintln("~password ________ Set the Password for ~login.")
		response += fmt.Sprintln("~login ___________ Login to Discord.")
		response += fmt.Sprintln("~logout __________ Logout from Discord.")
		response += fmt.Sprintln("~vregions ________ Display Voice Regions.")
		response += fmt.Sprintln("~vice ____________ Display Voice ICE information.")
		response += fmt.Sprintln("~gateway _________ Display websocket gateway.")
		response += fmt.Sprintln("~user ____________ Display logged in user information.")
		response += fmt.Sprintln("~user ID _________ Display user information for ID user.")
		response += fmt.Sprintln("~settings ________ Display the settings for the logged in user.")
		response += fmt.Sprintln("~guild ___________ Display active guild.")
		response += fmt.Sprintln("~guild ID ________ Set active guild to ID")
		response += fmt.Sprintln("~guilds __________ List all guilds")
		response += fmt.Sprintln("~members _________ List all mebers of current guild")
		response += fmt.Sprintln("~members ID ______ List all mebers of the id guild")
		response += fmt.Sprintln("~channel _________ Display active channel.")
		response += fmt.Sprintln("~channel ID ______ Set active channel to ID.")
		response += fmt.Sprintln("~channels ________ List all channels of bot has access to")
		response += fmt.Sprintln("~channels ID _____ List all channels of the id guild")
		response += fmt.Sprintln("~invites _________ List all invites for active guild")
		response += fmt.Sprintln("~gateway _________ Display Websocket Gateway.")
		response += fmt.Sprintln("~listen __________ Open Websocket and Listen for Events")
		response += fmt.Sprintln("~!listen _________ Close Websocket")
		response += fmt.Sprintln("~debug ___________ Toggle Debug on/off")
		response += fmt.Sprintln("~get URL _________ Test GET Request to URL")
		response += fmt.Sprintln("~quit ____________ Exit Bot\n")
		return
	}

	////////////////////////////////////////////////////////// Authentication

	// Authentication - Set the username for ~login
	if cmd == "username" {
		Username = cmdOpts
		response += "Done."
		return
	}

	// Authentication - Set the password for ~login
	if cmd == "password" {
		Password = cmdOpts
		response += "Done."
		return
	}

	// Authentication - Login to Discord
	if cmd == "login" {

		Session.Token, err = Session.Login(Username, Password)
		if err != nil {
			fmt.Println("Unable to login to Discord.")
			fmt.Println(err)
		}

		response += "Done."
		return
	}

	// Authentication - Logout from Discord
	if cmd == "logout" {
		err = Session.Logout()
		if err != nil {
			fmt.Println("Unable to logout from Discord.")
			fmt.Println(err)
		}
		response += "Done."
		return
	}

	////////////////////////////////////////////////////////// Voice

	// Voice - Display Voice Regions
	if cmd == "vregions" {
		r, err := Session.VoiceRegions()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(r)
		return
	}

	// Voice - Display Voice ICE information
	if cmd == "vice" {
		r, err := Session.VoiceICE()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(r)
		return
	}

	// temp command to test voice connection to channel
	if cmd == "vconnect" {
		Session.VoiceChannelJoin(Guild.ID, Channel.ID)
		return
	}

	// test playing audio!
	if cmd == "vplay" {
		playAudioFile("sample.mp3")
		return
	}

	////////////////////////////////////////////////////////// User

	// User - Display User Information
	if cmd == "user" {

		if cmdOpts == "" {
			cmdOpts = "@me"
		}

		user, err := Session.User(cmdOpts)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(user)
		response += "Done."
		return
	}

	// User - Display User's Settings
	if cmd == "settings" {

		if cmdOpts == "" {
			cmdOpts = "@me"
		}

		Settings, err := Session.UserSettings(cmdOpts)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(Settings)
		return
	}

	////////////////////////////////////////////////////////// Websocket

	// Websocket - Display Websocket Gateway
	if cmd == "gateway" {
		Gateway, err := Session.Gateway()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(Gateway)
		return
	}

	// Websocket - Open connection and listen for events.
	if cmd == "listen" {

		// open connection
		err = Session.Open()
		if err != nil {
			fmt.Println(err)
		}

		// Do Handshake? (dumb name)
		err = Session.Handshake()
		if err != nil {
			fmt.Println(err)
		}

		// Now listen for events / messages
		go Session.Listen()
		response += "Done."
		return

	}

	// Websocket - Close connection
	if cmd == "!listen" {
		Session.Close()
		response += "Done."
		return
	}

	////////////////////////////////////////////////////////// Guilds

	// Guilds - Set / Display Active Guild
	if cmd == "guild" {

		if cmdOpts == "" {
			printStruct(Guild)
			return
		}

		guilds, err := Session.UserGuilds("@me")
		if err != nil {
			fmt.Println(err)
		}

		for _, guild := range guilds {
			if guild.ID == cmdOpts {
				Guild = guild
				response += "Done."
				return
			}
		}

		response += "Could not find the requested Guild."
		return
	}

	// Guilds - List all guilds for user
	if cmd == "guilds" {

		if cmdOpts == "" {
			cmdOpts = "@me"
		}

		guilds, err := Session.UserGuilds(cmdOpts)
		if err != nil {
			response += fmt.Sprintln(err)
		}

		for _, guild := range guilds {
			response += fmt.Sprintf("%25s %s\n", guild.ID, guild.Name)
		}
		return
	}

	////////////////////////////////////////////////////////// Members

	// Members - Display memebers of a guild
	if cmd == "members" {

		if cmdOpts == "" {
			cmdOpts = Guild.ID
		}

		members, err := Session.GuildMembers(cmdOpts)
		if err != nil {
			fmt.Println(err)
		}
		for _, member := range members {
			response += fmt.Sprintf("%25s %s\n", member.User.ID, member.User.Username)
		}
		return
	}

	////////////////////////////////////////////////////////// Channels

	// Channel - Display / Set Active Channel
	if cmd == "channel" {

		if cmdOpts == "" {
			printStruct(Channel)
			return
		}

		// TODO: Loop over all Guilds
		channels, err := Session.GuildChannels(Guild.ID)
		if err != nil {
			fmt.Println(err)
		}

		for _, channel := range channels {

			if channel.ID == cmdOpts {
				Channel = channel
				response += "Done."
				return
			}
		}

		// Search Check Private Channels
		channels, err = Session.UserChannels("@me")
		if err != nil {
			fmt.Println(err)
		}

		for _, channel := range channels {

			if channel.ID == cmdOpts {
				Channel = channel
				response += "Done."
				return
			}
		}

		response += "That channel couldn't be found."
		return
	}

	// Channels - Display User Channels + Channels for the given Guild
	if cmd == "channels" {

		channels, err := Session.UserChannels("@me")
		for _, channel := range channels {
			response += fmt.Sprintf("%25s %s %t %s\n", channel.ID, channel.Type, channel.IsPrivate, channel.Recipient.Username)
		}

		if cmdOpts == "" {
			cmdOpts = Guild.ID
		}

		channels, err = Session.GuildChannels(cmdOpts)
		if err != nil {
			fmt.Println(err)
		}
		for _, channel := range channels {
			response += fmt.Sprintf("%25s %s %t %s\n", channel.ID, channel.Type, channel.IsPrivate, channel.Name)
		}
		return
	}

	// Channels - Send Message
	if cmd == "say" {
		Session.ChannelMessageSend(Channel.ID, cmdOpts)
		return
	}

	////////////////////////////////////////////////////////// Invites
	// not sure how these work yet.. Guild or Channel or just Invites :)

	// Invites - List all invites for a guild
	if cmd == "invites" {

		if cmdOpts == "" {
			cmdOpts = Guild.ID
		}

		invites, err := Session.GuildInvites(cmdOpts)
		if err != nil {
			fmt.Println(err)
		}
		for _, invite := range invites {
			response += fmt.Sprintln(invite)
		}
		return
	}

	////////////////////////////////////////////////////////// Testing

	// Enable / Disable Debug Logging
	if cmd == "debug" {

		if Session.Debug {
			Session.Debug = false
			response += "Debug off\n"
		} else {
			Session.Debug = true
			response += "Debug on\n"
		}
		return
	}

	// Testing - GET Request to URL
	if cmd == "get" {

		Session.Request("GET", cmdOpts, ``)
		response += "Done."
		return
	}

	// Testing - POST Request to URL
	if cmd == "post" {

		Session.Request("POST", cmdOpts, ``)
		response += "Done."
		return
	}

	if cmd == "delete" {

		Session.Request("DELETE", cmdOpts, ``)
		response += "Done."
		return
	}
	// Testing - URL Request
	if cmd == "request" {

		// format
		// ~request [TYPE] [URL] [DATA]
		// [TYPE] can be GET POST PATCH DELETE
		// [URL] is the full URL, no spaces
		// [DATA] is data to post.  Everything after URL is data.

		Session.Request("DELETE", cmdOpts, ``)
		response += "Done."
		return
	}
	////////////////////////////////////////////////////////// Other

	// Other - Quit program
	if cmd == "quit" {
		os.Exit(0)
	}

	// If we're still here....
	response += "I'm sorry I don't understand that command.  Try ~help"
	return
}

// prints json data in a easy to read format
func printJSON(body []byte) {
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, body, "", "\t")
	if error != nil {
		fmt.Print("JSON parse error: ", error)
	}
	fmt.Println(string(prettyJSON.Bytes()))
}

// I'm sure there's a better way...
func printStruct(i interface{}) error {

	json, err := json.Marshal(i)
	if err != nil {
		return err
	}
	printJSON(json)
	return err
}

func chk(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
