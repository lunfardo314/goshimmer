package packet

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"

	"github.com/iotaledger/hive.go/protocol/message"
	"github.com/iotaledger/hive.go/protocol/tlv"
)

var (
	// ErrInvalidMetricHeartbeat is returned for invalid Metric heartbeats.
	ErrInvalidMetricHeartbeat = errors.New("invalid Metric heartbeat")
)

var (
	// MetricHeartbeatMessageDefinition defines a metric heartbeat message's format.
	MetricHeartbeatMessageDefinition = &message.Definition{
		ID:             MessageTypeMetricHeartbeat,
		MaxBytesLength: 65535,
		VariableLength: true,
	}
)

// MetricHeartbeat represents a metric heartbeat packet.
type MetricHeartbeat struct {
	// The ID of the node who sent the heartbeat.
	// Must be contained when a heartbeat is serialized.
	OwnID       []byte
	OS          string
	Arch        string
	NumCPU      int
	CPUUsage    float64
	MemoryUsage uint64
}

// ParseMetricHeartbeat parses a slice of bytes (serialized packet) into a Metric heartbeat.
func ParseMetricHeartbeat(data []byte) (*MetricHeartbeat, error) {
	hb := &MetricHeartbeat{}

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	if err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(buf)
	err = decoder.Decode(hb)
	if err != nil {
		return nil, err
	}

	return hb, nil
}

// Bytes return the Metric heartbeat encoded as bytes
func (hb MetricHeartbeat) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(hb)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewMetricHeartbeatMessage serializes the given Metric heartbeat into a byte slice and adds a tlv header to the packet.
// message = tlv header + serialized packet
func NewMetricHeartbeatMessage(hb *MetricHeartbeat) ([]byte, error) {
	packet, err := hb.Bytes()
	if err != nil {
		return nil, err
	}

	// calculate total needed bytes based on packet
	packetSize := len(packet)

	// create a buffer for tlv header plus the packet
	buf := bytes.NewBuffer(make([]byte, 0, tlv.HeaderMessageDefinition.MaxBytesLength+uint16(packetSize)))
	// write tlv header into buffer
	if err := tlv.WriteHeader(buf, MessageTypeMetricHeartbeat, uint16(packetSize)); err != nil {
		return nil, err
	}
	// write serialized packet bytes into the buffer
	if err := binary.Write(buf, binary.BigEndian, packet); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}