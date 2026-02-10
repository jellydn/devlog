package natmsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
	"time"
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
	line := 42
	col := 10
	ts := mustParseTime(t, "2023-10-15T12:30:45Z")
	inputMsg := Message{
		Type:      "console",
		Level:     "error",
		Message:   "Test error message",
		URL:       "http://example.com",
		Timestamp: Timestamp{Time: ts},
		Source:    "app.js",
		Line:      &line,
		Column:    &col,
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
	if msg.Source != inputMsg.Source {
		t.Errorf("Source = %q, want %q", msg.Source, inputMsg.Source)
	}
	if msg.Line == nil || *msg.Line != line {
		t.Errorf("Line = %v, want %d", msg.Line, line)
	}
	if msg.Column == nil || *msg.Column != col {
		t.Errorf("Column = %v, want %d", msg.Column, col)
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

func TestTimestamp_UnmarshalJSON_String_RFC3339Nano(t *testing.T) {
	jsonData := `{"timestamp":"2023-10-15T12:30:45.123456789Z"}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := mustParseTime(t, "2023-10-15T12:30:45.123456789Z")
	if !msg.Timestamp.Time.Equal(expected) {
		t.Errorf("timestamp = %s, want %s", msg.Timestamp.Format(time.RFC3339Nano), expected.Format(time.RFC3339Nano))
	}
}

func TestTimestamp_UnmarshalJSON_String_RFC3339(t *testing.T) {
	jsonData := `{"timestamp":"2023-10-15T12:30:45Z"}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := mustParseTime(t, "2023-10-15T12:30:45Z")
	if !msg.Timestamp.Time.Equal(expected) {
		t.Errorf("timestamp = %s, want %s", msg.Timestamp.Format(time.RFC3339), expected.Format(time.RFC3339))
	}
}

func TestTimestamp_UnmarshalJSON_Number(t *testing.T) {
	// Unix timestamp in milliseconds: 1697371845123 = 2023-10-15T12:30:45.123Z
	jsonData := `{"timestamp":1697371845123}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMillis := int64(1697371845123)
	actualMillis := msg.Timestamp.UnixMilli()
	if actualMillis != expectedMillis {
		t.Errorf("timestamp millis = %d, want %d", actualMillis, expectedMillis)
	}
}

func TestTimestamp_UnmarshalJSON_Float(t *testing.T) {
	// Float timestamp in milliseconds
	jsonData := `{"timestamp":1697371845123.456}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMillis := int64(1697371845123)
	actualMillis := msg.Timestamp.UnixMilli()
	if actualMillis != expectedMillis {
		t.Errorf("timestamp millis = %d, want %d", actualMillis, expectedMillis)
	}
}

func TestTimestamp_UnmarshalJSON_InvalidString(t *testing.T) {
	jsonData := `{"timestamp":"not a valid timestamp"}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	err := json.Unmarshal([]byte(jsonData), &msg)
	if err == nil {
		t.Error("expected error for invalid timestamp string, got nil")
	}
}

func TestTimestamp_UnmarshalJSON_InvalidType(t *testing.T) {
	jsonData := `{"timestamp":true}`
	var msg struct {
		Timestamp Timestamp `json:"timestamp"`
	}

	err := json.Unmarshal([]byte(jsonData), &msg)
	if err == nil {
		t.Error("expected error for invalid timestamp type, got nil")
	}
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	ts := Timestamp{}
	ts.Time = mustParseTime(t, "2023-10-15T12:30:45.123456789Z")

	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `"2023-10-15T12:30:45.123456789Z"`
	if string(data) != expected {
		t.Errorf("marshaled = %s, want %s", string(data), expected)
	}
}

func TestMessage_LineColumn_Optional(t *testing.T) {
	// Test message without line/column
	jsonData := `{"type":"console","level":"log","message":"test","url":"http://example.com","timestamp":1697371845123}`

	var msg Message
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Line != nil {
		t.Errorf("Line should be nil, got %v", msg.Line)
	}
	if msg.Column != nil {
		t.Errorf("Column should be nil, got %v", msg.Column)
	}
}

func TestMessage_LineColumn_Present(t *testing.T) {
	// Test message with line/column
	jsonData := `{"type":"console","level":"log","message":"test","url":"http://example.com","timestamp":1697371845123,"line":42,"column":10}`

	var msg Message
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Line == nil || *msg.Line != 42 {
		t.Errorf("Line = %v, want 42", msg.Line)
	}
	if msg.Column == nil || *msg.Column != 10 {
		t.Errorf("Column = %v, want 10", msg.Column)
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return ts
}
