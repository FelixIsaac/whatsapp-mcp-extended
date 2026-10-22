package whatsapp

import (
	"encoding/binary"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestOrdinaryAudioIsNotAVoiceNote(t *testing.T) {
	for _, extension := range []string{"mp3", "m4a", "aac"} {
		t.Run(extension, func(t *testing.T) {
			mediaType, mimeType := mediaTypeAndMimeType("sample." + extension)
			if mediaType != whatsmeow.MediaAudio {
				t.Fatal("expected an audio upload")
			}
			audio := &waE2E.AudioMessage{
				PTT: proto.Bool(true), Seconds: proto.Uint32(30), Waveform: []byte{1},
			}
			if err := setAudioMessageMetadata(audio, mimeType, []byte("ordinary audio")); err != nil {
				t.Fatalf("ordinary audio must not require Ogg analysis: %v", err)
			}
			if audio.PTT == nil || audio.GetPTT() || audio.Seconds != nil || audio.Waveform != nil {
				t.Fatal("ordinary audio must have PTT false and no fabricated duration or waveform")
			}
		})
	}
}

func TestOggOpusRemainsAVoiceNote(t *testing.T) {
	// Synthetic Ogg pages exercise the existing parser without any connection.
	page := func(sequence uint32, granule uint64, payload []byte) []byte {
		data := make([]byte, 28+len(payload))
		copy(data, "OggS")
		binary.LittleEndian.PutUint64(data[6:14], granule)
		binary.LittleEndian.PutUint32(data[18:22], sequence)
		data[26], data[27] = 1, byte(len(payload))
		copy(data[28:], payload)
		return data
	}
	header := make([]byte, 19)
	copy(header, "OpusHead")
	header[8], header[9] = 1, 1
	binary.LittleEndian.PutUint32(header[12:16], 48000)
	data := append(page(0, 0, header), page(2, 96000, []byte{0xf8, 0xff, 0xfe})...)

	mediaType, mimeType := mediaTypeAndMimeType("sample.ogg")
	if mediaType != whatsmeow.MediaAudio {
		t.Fatal("expected an audio upload")
	}
	for _, mimeType := range []string{mimeType, "audio/opus"} {
		t.Run(mimeType, func(t *testing.T) {
			audio := &waE2E.AudioMessage{}
			if err := setAudioMessageMetadata(audio, mimeType, data); err != nil {
				t.Fatalf("voice-note analysis failed: %v", err)
			}
			if !audio.GetPTT() || audio.GetSeconds() != 2 || len(audio.GetWaveform()) != 64 {
				t.Fatal("Ogg Opus must retain analyzed duration, waveform and PTT true")
			}
		})
	}
}

func TestInvalidOggOpusCannotBecomeAVoiceNote(t *testing.T) {
	audio := &waE2E.AudioMessage{}
	if err := setAudioMessageMetadata(audio, "audio/ogg; codecs=opus", []byte("not Ogg")); err == nil {
		t.Fatal("expected the existing Ogg analysis to reject invalid input")
	}
	if audio.GetPTT() || audio.Seconds != nil || audio.Waveform != nil {
		t.Fatal("failed analysis must not produce voice-note metadata")
	}
}
