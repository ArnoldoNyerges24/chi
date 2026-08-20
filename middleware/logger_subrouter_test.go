package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSubrouterLoggerRoutePattern(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := RequestLogger(&DefaultLogFormatter{Logger: log.New(buf, "", 0)})

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(logger)
		r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/users/123")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	logOutput := buf.String()
	expectedPattern := `"/api/users/{id}"`
	if !strings.Contains(logOutput, expectedPattern) {
		t.Errorf("expected log output to contain route pattern %s, got: %s", expectedPattern, logOutput)
	}
}