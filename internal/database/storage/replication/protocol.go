package replication

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type SyncRequest struct {
	AfterSegment string
}

type SyncResponse struct {
	OK          bool
	SegmentName string
	SegmentData []byte
}

func Encode[T SyncRequest | SyncResponse](v *T) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

func Decode[T SyncRequest | SyncResponse](dst *T, data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(dst)
}
