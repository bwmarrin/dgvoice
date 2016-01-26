/*******************************************************************************
 * This is very experimental code and probably a long way from perfect or
 * ideal.  Please provide feed back on areas that would improve performance
 *
 */
package dgvoice

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

// NOTE: This API is not final and these are likely to change.
// Settings, these can be modified but they will not effect any
// currently running process.
var (
	// 1 for mono, 2 for stereo
	Channels int = 2

	// sample rate of frames, need to test valid options
	FrameRate int = 48000

	// Length of audio frame in ms can be 20, 40, or 60
	FrameTime int = 60
)

// Internal global vars.
// NOTE: This API is not final and these are likely to change.
var (
	opusEncoder *gopus.Encoder
	sequence    uint16
	timestamp   uint32
	run         *exec.Cmd
	FrameLength = func() int { return ((FrameRate / 1000) * FrameTime) } // Length of frame as uint16 array
	OpusMaxSize = func() int { return (FrameLength() * Channels * 2) }   // max size opus encoder can return
)

// init setups up the package for use :)
func init() {

	sequence = 0
	timestamp = 0
}

// KillPlayer forces the player to stop by killing the ffmpeg cmd process
// this method may be removed later in favor of using chans or bools to
// request a stop.
func KillPlayer() {
	run.Process.Kill()
}

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.
func PlayAudioFile(s *discordgo.Session, filename string) {

	opusMaxSize := OpusMaxSize()
	frameLength := FrameLength()
	frameRate := FrameRate
	channels := Channels

	// Create a shell command "object" to run.
	run = exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	stdout, err := run.StdoutPipe()
	if err != nil {
		fmt.Println("StdoutPipe Error:", err)
		return
	}

	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		fmt.Println("RunStart Error:", err)
		return
	}

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}

	// variables used during loop below
	audiobuf := make([]int16, frameLength*channels)

	// Send "speaking" packet over the voice websocket
	s.Voice.Speaking(true)
	// Send not "speaking" packet over the websocket when we finish
	defer s.Voice.Speaking(false)

	sendOpus := make(chan []byte, 5)
	go SendVoice(s, sendOpus)
	// TODO, check chan somehow to make sure it is ready?
	// can the chan be made inside SendVoice?

	for {

		// read data from ffmpeg stdout
		err = binary.Read(stdout, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			fmt.Println("Playback Error:", err)
			return
		}

		// try encoding ffmpeg frame with Opus
		opus, err := opusEncoder.Encode(audiobuf, frameLength, opusMaxSize)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		// send encoded opus data to the SendVoice channel
		sendOpus <- opus
	}
}

// SendVoice will listen on the given channel and send any
// pre-encoded opus audio to Discord.  Supposedly.
func SendVoice(s *discordgo.Session, buf <-chan []byte) {

	runtime.LockOSThread() // testing impact on quality

	opusMaxSize := OpusMaxSize()
	frameLength := FrameLength()
	frameTime := FrameTime

	// variables used during loop below
	udpPacket := make([]byte, opusMaxSize)
	opus := make([]byte, 1024)

	// build the parts that don't change in the udpPacket.
	udpPacket[0] = 0x80
	udpPacket[1] = 0x78
	binary.BigEndian.PutUint32(udpPacket[8:], s.Voice.OP2.SSRC)

	// start a send loop that loops until buf chan is closed
	ticker := time.NewTicker(time.Millisecond * time.Duration(frameTime))
	for {

		// Add sequence and timestamp to udpPacket
		binary.BigEndian.PutUint16(udpPacket[2:], sequence)
		binary.BigEndian.PutUint32(udpPacket[4:], timestamp)

		// Get data from chan and copy it into the udpPacket
		opus = <-buf
		copy(udpPacket[12:], opus)

		// block here until we're exactly at the right time :)
		// Then send rtp audio packet to Discord over UDP
		<-ticker.C
		s.Voice.UDPConn.Write(udpPacket[:(len(opus) + 12)])

		if (sequence) == 0xFFFF {
			sequence = 0
		} else {
			sequence += 1
		}

		if (timestamp + uint32(frameLength)) >= 0xFFFFFFFF {
			timestamp = 0
		} else {
			timestamp += uint32(frameLength)
		}
	}
}

// SendVoicePCM will listen on the given channel and send
// the PCM audio provided.
func SendVoicePCM(s *discordgo.Session, pcmc chan []int16) {

	var err error

	//	opusMaxSize := OpusMaxSize()
	//	frameLength := FrameLength()
	frameRate := FrameRate
	//	frameTime := FrameTime
	channels := Channels

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}
}
