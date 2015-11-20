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
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.

// Settings.
var (
	FrameRate   int = 48000                            // sample rate of frames
	FrameTime   int = 60                               // Length of audio frame in ms (20, 40, 60)
	FrameLength int = ((FrameRate / 1000) * FrameTime) // Length of frame as uint16 array
	OpusBitrate int = 96000                            // Bitrate to use when encoding
	OpusMaxSize int = (FrameLength * 2)                // max size opus encoder can return
)

func PlayAudioFile(s *discordgo.Session, filename string) {

	var sequence uint16 = 0  // used for voice play test
	var timestamp uint32 = 0 // used for voice play test

	opusEncoder, err := gopus.NewEncoder(FrameRate, 1, gopus.Audio)
	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}
	opusEncoder.SetBitrate(OpusBitrate)

	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(FrameRate), "-ac", "1", "pipe:1")
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
	udpPacket := make([]byte, OpusMaxSize)
	audiobuf := make([]int16, FrameLength)

	// build the parts that don't change in the udpPacket.
	udpPacket[0] = 0x80
	udpPacket[1] = 0x78
	binary.BigEndian.PutUint32(udpPacket[8:], s.Vop2.SSRC)

	// Send "speaking" packet over the voice websocket
	s.VoiceSpeaking()

	// start a read/encode/send loop that loops until EOF from ffmpeg
	ticker := time.NewTicker(time.Millisecond * time.Duration(FrameTime))
	for {
		// Add sequence and timestamp to udpPacket
		binary.BigEndian.PutUint16(udpPacket[2:], sequence)
		binary.BigEndian.PutUint32(udpPacket[4:], timestamp)

		// read 1920 bytes (960 int16) from ffmpeg stdout
		err = binary.Read(stdout, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Println("Reached EOF.")
			return
		}
		if err != nil {
			fmt.Println("Playback Error:", err)
			return
		}

		// try encoding ffmpeg frame with Opus
		opus, err := opusEncoder.Encode(audiobuf, FrameLength, OpusMaxSize)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		// copy opus result into udpPacket
		copy(udpPacket[12:], opus)

		// block here until we're exactly at the right time :)
		<-ticker.C

		// Send rtp audio packet to Discord over UDP
		s.UDPConn.Write(udpPacket[:(len(opus) + 12)])

		if (sequence) == 0xFFFF {
			sequence = 0
		} else {
			sequence += 1
		}

		if (timestamp + uint32(FrameLength)) >= 0xFFFFFFFF {
			timestamp = 0
		} else {
			timestamp += uint32(FrameLength)
		}
	}
}
