package telecom

import (
	"bufio"
	"encoding/binary"
	"io"
	"os/exec"
	"strconv"

	"github.com/b1naryth1ef/gopus"
	log "github.com/sirupsen/logrus"
)

type AvConvPlayable struct {
	bp     *BasicPlayable
	closed bool
	path   string
}

func NewAvConvPlayable(path string) *AvConvPlayable {
	playable := &AvConvPlayable{
		bp:     NewBasicPlayable(),
		closed: false,
		path:   path,
	}

	return playable
}

func (av *AvConvPlayable) Start() error {
	go av.runForever(av.path)
	return nil
}

func (av *AvConvPlayable) Output() (chan []byte, error) {
	return av.bp.Output()
}

func (av *AvConvPlayable) Close() {
	av.closed = true
	av.bp.Close()
}

func (av *AvConvPlayable) runForever(path string) {
	log.Infof("Starting AvConv playable for path %v", path)

	run := exec.Command(
		"ffmpeg",
		"-i", path,
		"-f", "s16le",
		"-ar", strconv.Itoa(frameRate),
		"-ac", strconv.Itoa(channels),
		"pipe:1",
	)
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		log.WithError(err).Error("Failed to open stdout pipe for ffmpeg")
		return
	}

	err = run.Start()
	if err != nil {
		log.WithError(err).Error("Failed to start ffmpeg process")
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)
	pcmOut := make(chan []int16, 2)

	go av.sendPCM(pcmOut)

	for {
		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			log.Warning("FFMPEG EOF")
			break
		} else if err != nil {
			log.WithError(err).Error("Unhandled ffmpeg error")
			break
		}

		if av.closed {
			log.Warning("AV Closed, exiting")
			break
		}

		pcmOut <- audiobuf
	}

	close(pcmOut)
	run.Wait()
}

func (av *AvConvPlayable) sendPCM(pcmIn <-chan []int16) {
	opusOut, _ := av.bp.Input()

	opusEncoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		log.WithError(err).Error("Failed to create new opus encoder")
		return
	}

	defer av.Close()

	for {
		recv, ok := <-pcmIn
		if !ok {
			return
		}

		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			log.WithError(err).Error("Failed to encode PCM data")
			return
		}

		opusOut <- opus
	}
}
