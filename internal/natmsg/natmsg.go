// Package natmsg implements the Native Messaging protocol for browser communication.
// The protocol uses length-prefixed JSON messages over stdin/stdout.
package natmsg

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Timestamp represents a flexible timestamp that can be unmarshaled from
// various JSON formats (string, number) and provides a time.Time value.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for Timestamp.
// Accepts both string (RFC3339) and numeric (Unix milliseconds) formats.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	// Try parsing as string first (RFC3339Nano format)
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		parsed, err := time.Parse(time.RFC3339Nano, s)
		if err == nil {
			t.Time = parsed
			return nil
		}
		// Try alternative format without nano precision
		parsed, err = time.Parse(time.RFC3339, s)
		if err == nil {
			t.Time = parsed
			return nil
		}
		return fmt.Errorf("invalid timestamp string format: %s", s)
	}

	// Try parsing as number (Unix milliseconds)
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		t.Time = time.UnixMilli(int64(n))
		return nil
	}

	return fmt.Errorf("timestamp must be a string or number")
}

// MarshalJSON implements custom JSON marshaling for Timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}

// Message represents a log message from the browser extension
type Message struct {
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	URL       string    `json:"url"`
	Timestamp Timestamp `json:"timestamp"`
	Source    string    `json:"source,omitempty"`
	Line      *int      `json:"line,omitempty"`
	Column    *int      `json:"column,omitempty"`
}

// UnmarshalJSON accepts line/column as either numbers or numeric strings.
func (m *Message) UnmarshalJSON(b []byte) error {
	type wireMessage struct {
		Type      string          `json:"type"`
		Level     string          `json:"level"`
		Message   string          `json:"message"`
		URL       string          `json:"url"`
		Timestamp Timestamp       `json:"timestamp"`
		Source    string          `json:"source,omitempty"`
		Line      json.RawMessage `json:"line,omitempty"`
		Column    json.RawMessage `json:"column,omitempty"`
	}

	var wire wireMessage
	if err := json.Unmarshal(b, &wire); err != nil {
		return err
	}

	line, err := parseOptionalInt(wire.Line)
	if err != nil {
		return fmt.Errorf("invalid line: %w", err)
	}
	column, err := parseOptionalInt(wire.Column)
	if err != nil {
		return fmt.Errorf("invalid column: %w", err)
	}

	m.Type = wire.Type
	m.Level = wire.Level
	m.Message = wire.Message
	m.URL = wire.URL
	m.Timestamp = wire.Timestamp
	m.Source = wire.Source
	m.Line = line
	m.Column = column

	return nil
}

func parseOptionalInt(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return &n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("must be an integer value")
		}
		return &n, nil
	}

	return nil, fmt.Errorf("must be a number or numeric string")
}

// Response represents a response message sent back to the browser
type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Host handles native messaging communication
type Host struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewHost creates a new native messaging host using stdin/stdout
func NewHost() *Host {
	return &Host{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// NewHostWithStreams creates a host with custom streams (useful for testing)
func NewHostWithStreams(reader io.Reader, writer io.Writer) *Host {
	return &Host{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
}

// ReadMessage reads a single message from the input stream.
// Returns the message or an error. io.EOF is returned when the stream ends.
func (h *Host) ReadMessage() (*Message, error) {
	// Read the 4-byte length prefix (uint32, native byte order)
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(h.reader, lengthBytes); err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	messageLen := binary.NativeEndian.Uint32(lengthBytes)
	if messageLen == 0 {
		return nil, fmt.Errorf("invalid message length: 0")
	}

	// Sanity check: messages shouldn't be larger than 10MB
	if messageLen > 10*1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", messageLen)
	}

	// Read the message body
	messageBytes := make([]byte, messageLen)
	if _, err := io.ReadFull(h.reader, messageBytes); err != nil {
		return nil, fmt.Errorf("failed to read message body: %w", err)
	}

	// Parse JSON
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message JSON: %w", err)
	}

	return &msg, nil
}

// WriteResponse sends a response message to the output stream.
// Used to acknowledge receipt or send errors back to the browser.
func (h *Host) WriteResponse(response Response) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Write length prefix
	lengthBytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(lengthBytes, uint32(len(data)))

	if _, err := h.writer.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write response length: %w", err)
	}

	if _, err := h.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

// SendAck sends a simple acknowledgment response
func (h *Host) SendAck(success bool, errMsg string) error {
	return h.WriteResponse(Response{Success: success, Error: errMsg})
}
