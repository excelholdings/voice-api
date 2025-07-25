package streaming

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"log"

	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/maxhawkins/go-webrtcvad"
)

// µ-law decoding lookup table
var muLawDecompressTable = [256]int16{
	-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
	-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
	-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
	-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
	-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
	-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
	-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
	-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
	-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
	-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
	-876, -844, -812, -780, -748, -716, -684, -652,
	-620, -588, -556, -524, -492, -460, -428, -396,
	-372, -356, -340, -324, -308, -292, -276, -260,
	-244, -228, -212, -196, -180, -164, -148, -132,
	-120, -112, -104, -96, -88, -80, -72, -64,
	-56, -48, -40, -32, -24, -16, -8, 0,
	32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
	23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
	15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
	11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
	7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
	5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
	3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
	2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
	1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
	1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
	876, 844, 812, 780, 748, 716, 684, 652,
	620, 588, 556, 524, 492, 460, 428, 396,
	372, 356, 340, 324, 308, 292, 276, 260,
	244, 228, 212, 196, 180, 164, 148, 132,
	120, 112, 104, 96, 88, 80, 72, 64,
	56, 48, 40, 32, 24, 16, 8, 0,
}

// muLawToPCM converts a mu-law encoded byte to a PCM value.
func muLawToPCM(muLaw byte) int16 {
	return muLawDecompressTable[muLaw]
}

// decodeMuLaw decodes a slice of mu-law encoded bytes to PCM.
func decodeMuLaw(data []byte) []int16 {
	pcm := make([]int16, len(data))
	for i, b := range data {
		pcm[i] = muLawToPCM(b)
	}
	return pcm
}

// resamplePCM resamples the PCM data to the desired output rate.
func resamplePCM(input []int16, inputRate, outputRate int) []int16 {
	ratio := float64(outputRate) / float64(inputRate)
	outputLength := int(float64(len(input)) * ratio)
	output := make([]int16, outputLength)
	for i := 0; i < outputLength; i++ {
		srcIndex := float64(i) / ratio
		srcIndexInt := int(srcIndex)
		srcIndexFrac := srcIndex - float64(srcIndexInt)
		if srcIndexInt+1 < len(input) {
			output[i] = int16(float64(input[srcIndexInt])*(1-srcIndexFrac) + float64(input[srcIndexInt+1])*srcIndexFrac)
		} else {
			output[i] = input[srcIndexInt]
		}
	}
	return output
}

// splitFrames splits the PCM data into frames of the specified duration.
func splitFrames(pcm []int16, frameSize int) [][]int16 {
	numFrames := (len(pcm) + frameSize - 1) / frameSize

	frames := make([][]int16, numFrames)
	for i := 0; i < numFrames; i++ {
		start := i * frameSize
		end := start + frameSize
		if end > len(pcm) {
			end = len(pcm)
		}
		frames[i] = pcm[start:end]
	}

	return frames
}

func (c *CallOrchestrator) handleWebRTC() {
	vad, err := webrtcvad.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := vad.SetMode(3); err != nil {
		log.Fatal(err)
	}

	speakingFrames := 0
	const frameDuration = 20 // 20 ms per frame
	const sampleRate = 16000
	const frameSize = sampleRate * frameDuration / 1000 // 320 samples per frame

	for {
		if c.done {
			break
		}
		mulawAudio := <-c.rtcAudioChan

		chunk, err := base64.StdEncoding.DecodeString(mulawAudio)
		if err != nil {
			logger.S.Error("Error decoding base64 payload:", err)
			continue
		}

		// Decode mu-law to PCM
		pcmData := decodeMuLaw(chunk)

		// Resample PCM data to 16000 Hz
		resampledData := resamplePCM(pcmData, 8000, sampleRate)

		// Split resampled data into frames of 320 samples (20 ms each)
		frames := splitFrames(resampledData, frameSize)

		// Process each frame
		for _, frame := range frames {
			// Ensure the frame is the correct size for the VAD
			if len(frame) != frameSize {
				logger.S.Error("Invalid frame size", "expected", frameSize, "got", len(frame))
				continue
			}

			// Convert frame to byte slice
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, frame)
			byteFrame := buf.Bytes()

			// Check if the frame is valid for the VAD
			if ok := vad.ValidRateAndFrameLength(sampleRate, len(byteFrame)/2); !ok {
				logger.S.Error("Invalid rate or frame length", "rate", sampleRate, "frame length", len(byteFrame)/2)
				continue
			}

			// Process the frame with the VAD
			active, err := vad.Process(sampleRate, byteFrame)
			if err != nil {
				log.Fatal(err)
			}

			// Track speaking frames
			if active {
				speakingFrames++
				//logger.S.Info("user talking")
			} else {
				speakingFrames = 0
			}

			// Check if user has been speaking for 500 ms
			if speakingFrames >= 460/frameDuration {
				//c.interruptionChan <- true
				logger.S.Info("detect user talking")
				speakingFrames = 0 // reset counter
			}
		}
	}
}
