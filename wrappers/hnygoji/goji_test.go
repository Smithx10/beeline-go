package hnygoji

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
	goji "goji.io"
	"goji.io/pat"
)

func TestGojiMiddleware(t *testing.T) {
	// set up libhoney to catch events instead of send them
	evCatcher := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "abcd",
		Dataset:  "efgh",
		Output:   evCatcher,
	})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello/pooh", nil)
	w := httptest.NewRecorder()

	// build the goji mux router with Middleware
	router := goji.NewMux()
	router.HandleFunc(pat.Get("/hello/:name"), func(_ http.ResponseWriter, _ *http.Request) {})
	router.Use(Middleware)
	// handle the request
	router.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Fields()
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
	name, ok := fields["goji.pat.name"]
	assert.True(t, ok, "goji.pat.name field must exist on middleware generated event")
	assert.Equal(t, "pooh", name, "successfully served request should have name var populated")

}
