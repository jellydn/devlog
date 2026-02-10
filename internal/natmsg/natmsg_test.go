package natmsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

func encodeMessage(t *testing.T, msg interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	// Prepend length
	lengthBytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(lengthBytes, uint32(len(data)))

	return append(lengthBytes, data...)
}

func TestHost_ReadMessage_Success(t *testing.T) {
	inputMsg := Message{
		Type:      "console",
		Level:     "error",
		Message:   "Test error message",
		URL:       "http://example.com",
		Timestamp: float64(1234567890),
		Source:    "app.js",
		Line:      float64(42),
		Column:    float64(10),
	}

	data := encodeMessage(t, inputMsg)
	reader := bytes.NewReader(data)
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	msg, err := host.ReadMessage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Type != inputMsg.Type {
		t.Errorf("Type = %q, want %q", msg.Type, inputMsg.Type)
	}
	if msg.Level != inputMsg.Level {
		t.Errorf("Level = %q, want %q", msg.Level, inputMsg.Level)
	}
	if msg.Message != inputMsg.Message {
		t.Errorf("Message = %q, want %q", msg.Message, inputMsg.Message)
	}
	if msg.URL != inputMsg.URL {
		t.Errorf("URL = %q, want %q", msg.URL, inputMsg.URL)
	}
	if fmt.Sprintf("%v", msg.Timestamp) != fmt.Sprintf("%v", inputMsg.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", msg.Timestamp, inputMsg.Timestamp)
	}
	if msg.Source != inputMsg.Source {
		t.Errorf("Source = %q, want %q", msg.Source, inputMsg.Source)
	}
	if fmt.Sprintf("%v", msg.Line) != fmt.Sprintf("%v", inputMsg.Line) {
		t.Errorf("Line = %v, want %v", msg.Line, inputMsg.Line)
	}
	if fmt.Sprintf("%v", msg.Column) != fmt.Sprintf("%v", inputMsg.Column) {
		t.Errorf("Column = %v, want %v", msg.Column, inputMsg.Column)
	}
}

func TestHost_ReadMessage_EOF(t *testing.T) {
	reader := bytes.NewReader([]byte{})
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	_, err := host.ReadMessage()
	if err != io.EOF {
		t.Errorf("expected EOF, got: %v", err)
	}
}

func TestHost_ReadMessage_InvalidLength(t *testing.T) {
	// Message with length 0
	data := []byte{0, 0, 0, 0}
	reader := bytes.NewReader(data)
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	_, err := host.ReadMessage()
	if err == nil {
		t.Error("expected error for zero-length message, got nil")
	}
}

func TestHost_ReadMessage_TooLarge(t *testing.T) {
	// Message claiming to be 100MB
	data := make([]byte, 4)
	binary.NativeEndian.PutUint32(data, 100*1024*1024)
	reader := bytes.NewReader(data)
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	_, err := host.ReadMessage()
	if err == nil {
		t.Error("expected error for oversized message, got nil")
	}
}

func TestHost_ReadMessage_InvalidJSON(t *testing.T) {
	// Valid length but invalid JSON
	msg := []byte("not valid json")
	data := make([]byte, 4)
	binary.NativeEndian.PutUint32(data, uint32(len(msg)))
	data = append(data, msg...)

	reader := bytes.NewReader(data)
	host := NewHostWithStreams(reader, &bytes.Buffer{})

	_, err := host.ReadMessage()
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestHost_WriteResponse(t *testing.T) {
	var output bytes.Buffer
	host := NewHostWithStreams(&bytes.Buffer{}, &output)

	response := Response{Success: true}

	if err := host.WriteResponse(response); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read length prefix
	lengthBytes := output.Bytes()[:4]
	length := binary.NativeEndian.Uint32(lengthBytes)

	// Read JSON
	jsonData := output.Bytes()[4 : 4+length]

	var decoded Response
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if decoded.Success != true {
		t.Errorf("success = %v, want true", decoded.Success)
	}
}

func TestHost_SendAck_Success(t *testing.T) {
	var output bytes.Buffer
	host := NewHostWithStreams(&bytes.Buffer{}, &output)

	if err := host.SendAck(true, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response format
	lengthBytes := output.Bytes()[:4]
	length := binary.NativeEndian.Uint32(lengthBytes)
	jsonData := output.Bytes()[4 : 4+length]

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to decode ack: %v", err)
	}

	if decoded["success"] != true {
		t.Errorf("success = %v, want true", decoded["success"])
	}
	if _, hasError := decoded["error"]; hasError {
		t.Error("unexpected error field in success ack")
	}
}

func TestHost_SendAck_Error(t *testing.T) {
	var output bytes.Buffer
	host := NewHostWithStreams(&bytes.Buffer{}, &output)

	if err := host.SendAck(false, "something went wrong"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response format
	lengthBytes := output.Bytes()[:4]
	length := binary.NativeEndian.Uint32(lengthBytes)
	jsonData := output.Bytes()[4 : 4+length]

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to decode ack: %v", err)
	}

	if decoded["success"] != false {
		t.Errorf("success = %v, want false", decoded["success"])
	}
	if decoded["error"] != "something went wrong" {
		t.Errorf("error = %q, want %q", decoded["error"], "something went wrong")
	}
}
