package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jellydn/devlog/internal/natmsg"
)

func encodeNativeMessage(t *testing.T, msg interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}
	lengthBytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(lengthBytes, uint32(len(data)))
	return append(lengthBytes, data...)
}

func decodeAck(t *testing.T, data []byte) natmsg.Response {
	t.Helper()
	if len(data) < 4 {
		t.Fatalf("ack too short: %d bytes", len(data))
	}
	length := binary.NativeEndian.Uint32(data[:4])
	if int(length) > len(data)-4 {
		t.Fatalf("ack length %d exceeds remaining %d bytes", length, len(data)-4)
	}
	var resp natmsg.Response
	if err := json.Unmarshal(data[4:4+length], &resp); err != nil {
		t.Fatalf("failed to unmarshal ack: %v", err)
	}
	return resp
}

func sampleMessage(level, message string) natmsg.Message {
	return natmsg.Message{
		Type:      "console",
		Level:     level,
		Message:   message,
		URL:       "http://localhost:3000/",
		Timestamp: natmsg.Timestamp{Time: time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)},
	}
}

func TestRun_RequiresLogPath(t *testing.T) {
	var stderr bytes.Buffer
	err := run(nil, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatal("expected error when log path is missing")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("stderr should include usage, got %q", stderr.String())
	}
}

func TestRun_ProcessesMessagesAndFiltersLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	var stdin bytes.Buffer
	stdin.Write(encodeNativeMessage(t, sampleMessage("error", "boom")))
	stdin.Write(encodeNativeMessage(t, sampleMessage("log", "hello")))
	stdin.Write(encodeNativeMessage(t, sampleMessage("warn", "careful")))

	var stdout, stderr bytes.Buffer
	err := run([]string{logPath, "error", "warn"}, &stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() unexpected error: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logText := string(content)
	if !strings.Contains(logText, "boom") {
		t.Errorf("expected error message in log, got %q", logText)
	}
	if !strings.Contains(logText, "careful") {
		t.Errorf("expected warn message in log, got %q", logText)
	}
	if strings.Contains(logText, "hello") {
		t.Errorf("log-level message should be filtered out, got %q", logText)
	}

	// Two acks for error+warn (filtered log still gets acked after successful Log call)
	// Actually logger.Log returns nil for filtered messages, so all three get success acks.
	out := stdout.Bytes()
	ackCount := 0
	for len(out) >= 4 {
		length := binary.NativeEndian.Uint32(out[:4])
		if int(length)+4 > len(out) {
			break
		}
		resp := decodeAck(t, out[:4+length])
		if !resp.Success {
			t.Errorf("expected success ack, got %#v", resp)
		}
		ackCount++
		out = out[4+length:]
	}
	if ackCount != 3 {
		t.Errorf("ack count = %d, want 3", ackCount)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty, got %q", stderr.String())
	}
}

func TestRun_EOFExitsCleanly(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	var stdout, stderr bytes.Buffer
	err := run([]string{logPath}, bytes.NewReader(nil), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() on empty stdin should return nil, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no acks on empty stdin, got %q", stdout.String())
	}
}

func TestRun_MalformedMessageContinues(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "browser.log")

	var stdin bytes.Buffer
	// Invalid JSON body with non-zero length
	bad := []byte(`{not-json`)
	lengthBytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(lengthBytes, uint32(len(bad)))
	stdin.Write(lengthBytes)
	stdin.Write(bad)
	// Followed by a valid message that should still be processed
	stdin.Write(encodeNativeMessage(t, sampleMessage("error", "recovered")))

	var stdout, stderr bytes.Buffer
	err := run([]string{logPath}, &stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() unexpected error: %v", err)
	}

	if !strings.Contains(stderr.String(), "Error reading message:") {
		t.Errorf("expected read error on stderr, got %q", stderr.String())
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "recovered") {
		t.Errorf("valid message after malformed one should be logged, got %q", content)
	}

	// First ack should fail, second succeed
	out := stdout.Bytes()
	var responses []natmsg.Response
	for len(out) >= 4 {
		length := binary.NativeEndian.Uint32(out[:4])
		if int(length)+4 > len(out) {
			break
		}
		responses = append(responses, decodeAck(t, out[:4+length]))
		out = out[4+length:]
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 acks, got %d", len(responses))
	}
	if responses[0].Success {
		t.Error("first ack should report failure for malformed message")
	}
	if !responses[1].Success {
		t.Error("second ack should report success")
	}
}

func TestRun_LoggerCreateFailure(t *testing.T) {
	// Use a path under a file so MkdirAll/OpenFile fails
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}
	badPath := filepath.Join(blocker, "browser.log")

	var stderr bytes.Buffer
	err := run([]string{badPath}, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatal("expected error when logger cannot be created")
	}
	if !strings.Contains(err.Error(), "failed to create logger") {
		t.Errorf("error = %q, want create logger failure", err.Error())
	}
}

type failingLogger struct{}

func (f failingLogger) Log(msg *natmsg.Message) error {
	return errors.New("disk full")
}

func TestProcessMessages_LogWriteErrorSendsFailureAck(t *testing.T) {
	msg := sampleMessage("error", "write-me")
	data := encodeNativeMessage(t, msg)

	var stdout, stderr bytes.Buffer
	host := natmsg.NewHostWithStreams(bytes.NewReader(data), &stdout)
	err := processMessages(failingLogger{}, host, &stderr)
	if err != nil {
		t.Fatalf("processMessages() unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Error writing log:") {
		t.Errorf("expected write error on stderr, got %q", stderr.String())
	}
	resp := decodeAck(t, stdout.Bytes())
	if resp.Success {
		t.Error("expected failure ack when Log returns error")
	}
	if !strings.Contains(resp.Error, "disk full") {
		t.Errorf("ack error = %q, want disk full", resp.Error)
	}
}

func TestProcessMessages_EmptyInputIsEOF(t *testing.T) {
	host := natmsg.NewHostWithStreams(bytes.NewReader(nil), &bytes.Buffer{})
	if err := processMessages(failingLogger{}, host, io.Discard); err != nil {
		t.Fatalf("empty input should be clean EOF, got %v", err)
	}
}
