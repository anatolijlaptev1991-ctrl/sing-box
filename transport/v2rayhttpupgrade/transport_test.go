package v2rayhttpupgrade

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

const (
	testPath    = "/upgrade"
	testTimeout = 5 * time.Second
)

// testPayload starts with 0x00 followed by a UUID-like tail, the shape of a
// VLESS request header whose first byte was lost in the original bug report.
var testPayload = []byte{
	0x00, 0xb8, 0x31, 0x38, 0x1d, 0x6a, 0x5c, 0x4e, 0x8f, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd,
	0xef, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
}

type readResult struct {
	data []byte
	err  error
}

type captureHandler struct {
	size    int
	results chan readResult
}

func newCaptureHandler(size int) *captureHandler {
	return &captureHandler{size: size, results: make(chan readResult, 1)}
}

func (h *captureHandler) NewConnectionEx(ctx context.Context, conn net.Conn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	defer conn.Close()
	// Shorter than the wait in requireServerReceived, so a lost byte is reported as a short read.
	_ = conn.SetReadDeadline(time.Now().Add(testTimeout / 2))
	data := make([]byte, h.size)
	n, err := io.ReadFull(conn, data)
	h.results <- readResult{data: data[:n], err: err}
}

// delayedHijackWriter keeps the response writer between "101 flushed" and
// "connection hijacked" open for a while, which is the window in which the
// net/http background read picks up the first byte the client sends.
type delayedHijackWriter struct {
	http.ResponseWriter
	delay time.Duration
}

func (w *delayedHijackWriter) FlushError() error {
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *delayedHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	time.Sleep(w.delay)
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func startTestServer(t *testing.T, handler *captureHandler, hijackDelay time.Duration) string {
	t.Helper()
	server, err := NewServer(context.Background(), logger.NOP(), option.V2RayHTTPUpgradeOptions{Path: testPath}, nil, handler)
	require.NoError(t, err)
	if hijackDelay > 0 {
		inner := server.httpServer.Handler
		server.httpServer.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			inner.ServeHTTP(&delayedHijackWriter{ResponseWriter: writer, delay: hijackDelay}, request)
		})
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go server.Serve(listener)
	t.Cleanup(func() {
		server.Close()
	})
	return listener.Addr().String()
}

func upgradeRequest(addr string) []byte {
	return []byte("GET " + testPath + " HTTP/1.1\r\nHost: " + addr + "\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
}

func requireServerReceived(t *testing.T, handler *captureHandler, expected []byte) {
	t.Helper()
	select {
	case result := <-handler.results:
		require.NoError(t, result.err, "server read %d of %d bytes: % x", len(result.data), len(expected), result.data)
		require.Equal(t, expected, result.data)
	case <-time.After(testTimeout):
		t.Fatal("server handler was never called")
	}
}

func dialRaw(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, testTimeout)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(testTimeout)))
	return conn
}

func TestServerKeepsBytesSentRightAfter101(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name        string
		hijackDelay time.Duration
	}{
		// The byte arrives while the handler is between the flushed 101 and Hijack.
		{name: "before_hijack", hijackDelay: 200 * time.Millisecond},
		{name: "plain"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			handler := newCaptureHandler(len(testPayload))
			addr := startTestServer(t, handler, testCase.hijackDelay)
			conn := dialRaw(t, addr)
			_, err := conn.Write(upgradeRequest(addr))
			require.NoError(t, err)
			response, err := http.ReadResponse(bufio.NewReader(conn), nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
			_, err = conn.Write(testPayload)
			require.NoError(t, err)
			requireServerReceived(t, handler, testPayload)
		})
	}
}

func TestServerKeepsBytesPipelinedWithRequest(t *testing.T) {
	t.Parallel()
	handler := newCaptureHandler(len(testPayload))
	addr := startTestServer(t, handler, 0)
	conn := dialRaw(t, addr)
	_, err := conn.Write(append(upgradeRequest(addr), testPayload...))
	require.NoError(t, err)
	requireServerReceived(t, handler, testPayload)
}

func TestClientKeepsBytesSentRightAfter101(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(testTimeout))
		_, readErr := http.ReadRequest(bufio.NewReader(conn))
		if readErr != nil {
			return
		}
		// One write: the 101 and the first payload bytes land in the client's read buffer together.
		response := []byte("HTTP/1.1 101 Switching Protocols\r\nConnection: upgrade\r\nUpgrade: websocket\r\n\r\n")
		_, _ = conn.Write(append(response, testPayload...))
		_, _ = io.Copy(io.Discard, conn)
	}()
	client, err := NewClient(context.Background(), N.SystemDialer, M.ParseSocksaddr(listener.Addr().String()), option.V2RayHTTPUpgradeOptions{Path: testPath}, nil)
	require.NoError(t, err)
	conn, err := client.DialContext(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(testTimeout)))
	received := make([]byte, len(testPayload))
	n, err := io.ReadFull(conn, received)
	require.NoError(t, err, "client read %d of %d bytes", n, len(testPayload))
	require.True(t, bytes.Equal(testPayload, received), "got % x", received)
}
