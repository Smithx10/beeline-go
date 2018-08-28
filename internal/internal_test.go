package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/websocket"
)

func TestParseTraceHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Amzn-Trace-Id", "Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app")
	ev := libhoney.NewEvent()
	parseTraceHeader(req, ev)
	fs := ev.Fields()
	// spew.Dump(fs)
	assert.Equal(t, "1-67891234-12456789abcdef012345678", fs["request.header.aws_trace_id.Self"])
	assert.Equal(t, "1-67891233-abcdef012345678912345678", fs["request.header.aws_trace_id.Root"])
	assert.Equal(t, "app", fs["request.header.aws_trace_id.CalledFrom"])
}

func TestHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Host", "example.com")
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "example.com", fs["request.host"])
}

func TestURLHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "example.com", fs["request.host"])
}

func TestHijackingWebsockets(t *testing.T) {
	clientSent := "helloWorld"
	srvReceived := ""
	srvClose := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Comment out this line to see the working behaviour without the internal.ResponseWriter.
		w = NewResponseWriter(w)
		defer func() {
			srvClose <- struct{}{}
		}()

		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error("upgrade:", err)
			return
		}
		defer c.Close()
		_, message, err := c.ReadMessage()
		if err != nil {
			t.Log("read:", err)
			return
		}
		srvReceived = string(message)
	}))
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):]
	t.Logf("url: %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	conn.WriteMessage(websocket.TextMessage, []byte(clientSent))
	<-srvClose

	if clientSent != srvReceived {
		t.Errorf("sent %s but received %s", clientSent, srvReceived)
	}
}
