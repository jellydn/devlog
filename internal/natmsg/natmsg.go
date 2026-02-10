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
)

// Message represents a log message from the browser extension
type Message struct {
	Type      string `json:"type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	URL       string `json:"url"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
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
func (h *Host) WriteResponse(response map[string]interface{}) error {
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
	response := map[string]interface{}{
		"success": success,
	}
	if errMsg != "" {
		response["error"] = errMsg
	}
	return h.WriteResponse(response)
}
