package xifer

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSubscribeStatusRedactsTokenInLogs(t *testing.T) {
	messageReceived := make(chan []byte, 1)
	server := newWebsocketTestServer(t, func(conn *websocket.Conn) {
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		messageReceived <- message
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewWSClient("ws"+strings.TrimPrefix(server.URL, "http"), logger)

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	originalStdout := os.Stdout
	os.Stdout = stdoutWriter
	var stdoutOutput bytes.Buffer
	stdoutDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&stdoutOutput, stdoutReader)
		stdoutDone <- copyErr
	}()
	defer func() {
		os.Stdout = originalStdout
		_ = stdoutWriter.Close()
		_ = stdoutReader.Close()
	}()

	if err := client.ConnectAndReadEvents(context.Background(), SubscribeStatus("apikey|SECRET", "account-1", true, false)); err != nil {
		t.Fatalf("ConnectAndReadEvents: %v", err)
	}
	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	os.Stdout = originalStdout
	if err := <-stdoutDone; err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	_ = stdoutReader.Close()

	select {
	case message := <-messageReceived:
		if !bytes.Contains(message, []byte(`"token":"apikey|SECRET"`)) {
			t.Fatalf("server received subscription without expected token: %s", message)
		}
	default:
		t.Fatal("server did not receive subscription")
	}
	if strings.Contains(logOutput.String(), "SECRET") {
		t.Fatalf("logs contain the token: %s", logOutput.String())
	}
	if !strings.Contains(logOutput.String(), "level=DEBUG") || !strings.Contains(logOutput.String(), "[REDACTED]") {
		t.Fatalf("debug logs do not include redacted subscription: %s", logOutput.String())
	}
	if strings.Contains(stdoutOutput.String(), "SECRET") {
		t.Fatalf("stdout contains the token: %s", stdoutOutput.String())
	}
}

func TestNewWSClientNilLoggerCanWrite(t *testing.T) {
	messageReceived := make(chan []byte, 1)
	server := newWebsocketTestServer(t, func(conn *websocket.Conn) {
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read message: %v", err)
			return
		}
		messageReceived <- message
	})
	defer server.Close()

	client := NewWSClient("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	conn, _, err := websocket.DefaultDialer.Dial(client.uri, nil)
	if err != nil {
		t.Fatalf("dial test websocket: %v", err)
	}
	defer func() { _ = conn.Close() }()
	client.conn = conn

	if err := client.write(messageSubscribe{Type: subscribe, Channel: heartbeat}); err != nil {
		t.Fatalf("write with nil logger: %v", err)
	}
	select {
	case <-messageReceived:
	case <-time.After(time.Second):
		t.Fatal("server did not receive message")
	}
}

func newWebsocketTestServer(t *testing.T, handle func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		handle(conn)
	}))
}
