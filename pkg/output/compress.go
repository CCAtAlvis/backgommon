package output

import (
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
)

// WriteZSTD compresses data with zstd (default speed) and writes path.
func WriteZSTD(path string, data []byte) error {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return fmt.Errorf("zstd encoder: %w", err)
	}
	defer enc.Close()
	compressed := enc.EncodeAll(data, make([]byte, 0, len(data)/2))
	return os.WriteFile(path, compressed, 0644)
}

// ReadZSTD reads and decompresses a zstd file.
func ReadZSTD(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decoder: %w", err)
	}
	defer dec.Close()
	return dec.DecodeAll(raw, nil)
}
