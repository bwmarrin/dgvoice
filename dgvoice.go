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
	"sync"
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
	FrameTime int = 20

	send    bool
	sendpcm bool
	mu      sync.Mutex
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

	// variables used during loop below
	audiobuf := make([]int16, frameLength*channels)

	// Send "speaking" packet over the voice websocket
	s.Voice.Speaking(true)
	// Send not "speaking" packet over the websocket when we finish
	defer s.Voice.Speaking(false)

	sendPCM := make(chan []int16, 2)
	defer close(sendPCM)
	go SendPCM(s, sendPCM)
	// TODO, check chan somehow to make sure it is ready?
	// can the chan be made inside Send?

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

		// Send received PCM to the sendPCM channel
		sendPCM <- audiobuf
	}
}

// SendPCM will listen on the given channel and send any
// PCM audio to Discord.  Supposedly.
func SendPCM(s *discordgo.Session, pcm <-chan []int16) {

	// Temp hacky stuff to make sure this only runs one instance at a time.
	mu.Lock()
	if sendpcm {
		// some seriously hacky stuff here
		time.Sleep(1 * time.Second)
		if sendpcm {
			mu.Unlock()
			return
		}
	}
	sendpcm = true
	mu.Unlock()

	defer func() {
		sendpcm = false
	}()

	opusMaxSize := OpusMaxSize()
	frameLength := FrameLength()
	frameRate := FrameRate
	channels := Channels
	frameTime := FrameTime

	var err error
	var ok bool

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}

	sendOpus := make(chan []byte, 2)
	defer close(sendOpus)
	go Send(s, frameTime, opusMaxSize, frameLength, sendOpus)
	// TODO, check chan somehow to make sure it is ready?
	// can the chan be made inside Send?

	audiobuf := make([]int16, frameLength*channels)
	for {

		// read pcm from chan
		audiobuf, ok = <-pcm
		if !ok {
			return
		}

		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(audiobuf, frameLength, opusMaxSize)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		// send encoded opus data to the sendOpus channel
		sendOpus <- opus
	}
}

// Send will listen on the given channel and send any
// pre-encoded opus audio to Discord.  Supposedly.
func Send(s *discordgo.Session, frameTime, opusMaxSize, frameLength int, buf <-chan []byte) {

	// Temp hacky stuff to make sure this only runs one instance at a time.
	mu.Lock()
	if send {
		// some seriously hacky stuff here
		time.Sleep(1 * time.Second)
		if send {
			mu.Unlock()
			return
		}
	}
	send = true
	mu.Unlock()

	defer func() {
		send = false
	}()

	runtime.LockOSThread() // testing impact on quality

	// variables used during loop below
	udpPacket := make([]byte, opusMaxSize)
	opus := make([]byte, 1024)
	var ok bool = true

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
		opus, ok = <-buf
		if !ok {
			return
		}
		// TODO: Is there a better way to avoid all this
		// data coping?
		copy(udpPacket[12:], opus)

		// block here until we're exactly at the right time :)
		// Then send rtp audio packet to Discord over UDP
		<-ticker.C
		s.Voice.UDPConn.Write(udpPacket[:12+(len(opus))])

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
