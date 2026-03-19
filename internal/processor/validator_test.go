package processor

import (
	"bytes"
	"testing"
)

func TestValidateMagicBytes_ValidMP4(t *testing.T) {
	// Typical MP4 header: 4 bytes size + "ftyp" + subtype
	header := make([]byte, 12)
	header[0] = 0x00
	header[1] = 0x00
	header[2] = 0x00
	header[3] = 0x1C
	copy(header[4:8], "ftyp")
	copy(header[8:12], "isom")

	err := ValidateMagicBytes(bytes.NewReader(header))
	if err != nil {
		t.Errorf("expected valid MP4, got error: %v", err)
	}
}

func TestValidateMagicBytes_InvalidFile(t *testing.T) {
	data := []byte("this is not a video file at all")
	err := ValidateMagicBytes(bytes.NewReader(data))
	if err != ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got: %v", err)
	}
}

func TestValidateMagicBytes_TooSmall(t *testing.T) {
	data := []byte("tiny")
	err := ValidateMagicBytes(bytes.NewReader(data))
	if err != ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got: %v", err)
	}
}

func TestValidateSize_OK(t *testing.T) {
	err := ValidateSize(100, 500)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSize_TooLarge(t *testing.T) {
	err := ValidateSize(600, 500)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}
