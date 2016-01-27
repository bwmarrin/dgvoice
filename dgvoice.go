/*******************************************************************************
 * This is very experimental code and probably a long way from perfect or
 * ideal.  Please provide feed back on areas that would improve performance
 *
 */

// package dgvoice provides opus encoding and audio file playback for the
// Discordgo package.
package dgvoice

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

// NOTE: This API is not final and these are likely to change.

// Technically the below settings can be adjusted however that poses
// a lot of other problems that are not handled well at this time.
// These below values seem to provide the best overall performance
const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

var (
	opusEncoder *gopus.Encoder
	run         *exec.Cmd
	sendpcm     bool
	mu          sync.Mutex
	PCM         chan []int16
)

func init() {
	PCM = make(chan []int16, 2)
}

// SendPCM will listen on the given channel and send any
// PCM audio to Discord.  Supposedly :)
func SendPCM(v *discordgo.Voice, pcm <-chan []int16) {

	// make sure this only runs one instance at a time.
	mu.Lock()
	if sendpcm || pcm == nil {
		mu.Unlock()
		return
	}
	sendpcm = true
	mu.Unlock()

	defer func() { sendpcm = false }()

	var err error

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recvbuf, ok := <-pcm
		if !ok {
			return
		}

		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(recvbuf, frameSize, maxBytes)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		// send encoded opus data to the sendOpus channel
		v.Opus <- opus
	}
}

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.
func PlayAudioFile(s *discordgo.Session, filename string) {

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

	// buffer used during loop below
	audiobuf := make([]int16, frameSize*channels)

	// Send "speaking" packet over the voice websocket
	s.Voice.Speaking(true)

	// Send not "speaking" packet over the websocket when we finish
	defer s.Voice.Speaking(false)

	// will actually only spawn one instance, a bit hacky.
	go SendPCM(s.Voice, PCM)

	for {

		// read data from ffmpeg stdout
		err = binary.Read(stdout, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Println("Reached EOF")
			return
		}
		if err != nil {
			fmt.Println("error reading from ffmpeg stdout :", err)
			return
		}

		// Send received PCM to the sendPCM channel
		PCM <- audiobuf
	}
}

// KillPlayer forces the player to stop by killing the ffmpeg cmd process
// this method may be removed later in favor of using chans or bools to
// request a stop.
func KillPlayer() {
	run.Process.Kill()
}
