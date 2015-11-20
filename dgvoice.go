package dgvoice

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

func playAudioFile(s *discordgo.Session, filename string) {

	var sequence uint16 = 0  // used for voice play test
	var timestamp uint32 = 0 // used for voice play test

	opusEncoder, err := gopus.NewEncoder(48000, 1, gopus.Audio)
	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}
	opusEncoder.SetBitrate(96000)

	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", "48000", "-ac", "1", "pipe:1")
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
	udpPacket := make([]byte, 4000)
	audiobuf := make([]int16, 960)

	// build the parts that don't change in the udpPacket.
	udpPacket[0] = 0x80
	udpPacket[1] = 0x78
	binary.BigEndian.PutUint32(udpPacket[8:], s.Vop2.SSRC)

	// Send "speaking" packet over the voice websocket
	s.VoiceSpeaking()

	// start a 20ms read/encode/send loop that loops until EOF from ffmpeg
	ticker := time.NewTicker(time.Millisecond * 20)
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
		opus, err := opusEncoder.Encode(audiobuf, 960, 1920)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		// copy opus result into udpPacket
		copy(udpPacket[12:], opus)

		<-ticker.C // block here until we're exactly at 20ms
		s.UDPConn.Write(udpPacket[:(len(opus) + 12)])

		// increment sequence and timestamp
		// timestamp should be calculated based on something.. :)
		if (sequence) == 0xFFFF {
			sequence = 0
		} else {
			sequence += 1 // this just increments each loop
		}

		if (timestamp + 960) >= 0xFFFFFFFF {
			timestamp = 0
		} else {
			timestamp += 960 // also just increments each loop
		}

	}
}
