/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package remotecommand

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/httpstream/wsstream"
)

// TestCreateWebSocketStreams_ServerNotReady verifies that when the WebSocket
// server side terminates during the handshake (simulating the
// ErrWebSocketServerNotReady scenario), createWebSocketStreams returns nil,false.
func TestCreateWebSocketStreams_ServerNotReady(t *testing.T) {
	// minimal websocket upgrade request headers
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Protocol", wsstream.ChannelWebSocketProtocol)

	// a ResponseWriter that hijacks and immediately closes the server side.
	w := &immediateCloseHijackerResponseWriter{ResponseRecorder: httptest.NewRecorder()}

	opts := &Options{Stdin: false, Stdout: false, Stderr: false, TTY: false}

	ctx, ok := createWebSocketStreams(req, w, opts, 1*time.Second)
	if ctx != nil || ok {
		t.Fatalf("expected createWebSocketStreams to return nil,false when server dies during handshake, got ctx=%v ok=%v", ctx, ok)
	}
}

// immediateCloseHijackerResponseWriter implements http.ResponseWriter and http.Hijacker.
// Its Hijack returns a connection whose server side is closed immediately to simulate
// a WebSocket server that finishes before becoming ready.
type immediateCloseHijackerResponseWriter struct {
	*httptest.ResponseRecorder
}

func (w *immediateCloseHijackerResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	clientConn, serverConn := net.Pipe()
	// simulate server termination
	_ = serverConn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	return clientConn, rw, nil
}

var _ http.Hijacker = &immediateCloseHijackerResponseWriter{}
