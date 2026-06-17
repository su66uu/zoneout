package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"zoneout/internal/playback"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const agentAddr = "127.0.0.1:17777"

type playRequest struct {
	StreamURL string `json:"stream_url"`
}

type statusResponse struct {
	State     string `json:"state"`
	StreamUrl string `json:"stream_url,omitempty"`
	Error     string `json:"error,omitempty"`
}

type playbackState string

const (
	stateIdle       playbackState = "idle"
	stateConnecting playbackState = "connecting"
	statePlaying    playbackState = "playing"
	stateError      playbackState = "error"
)

type agentState struct {
	mu        sync.Mutex
	state     playbackState
	streamURL string
	err       string
	cancel    context.CancelFunc
}

var state = &agentState{state: "idle"}

func main() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		_, _ = w.Write([]byte("ok"))
	})

	router.Get("/status", statusHandler)
	router.Post("/play", playMusicHandler)
	router.Post("/stop", stopMusicHandler)

	log.Printf("Starting the agent server at %s", agentAddr)
	if err := http.ListenAndServe(agentAddr, router); err != nil {
		log.Fatal(err)
	}
}

func playMusicHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody playRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.StreamURL == "" {
		http.Error(w, "stream_url is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	state.mu.Lock()
	if state.cancel != nil {
		state.cancel()
	}

	state.state = stateConnecting
	state.streamURL = reqBody.StreamURL
	state.err = ""
	state.cancel = cancel
	state.mu.Unlock()

	go func() {
		state.mu.Lock()
		state.state = statePlaying
		state.mu.Unlock()

		err := playback.Play(ctx, reqBody.StreamURL)

		state.mu.Lock()
		defer state.mu.Unlock()

		if errors.Is(err, context.Canceled) {
			state.state = stateIdle
			state.err = ""
			return
		}

		if err != nil {
			state.state = stateError
			state.err = err.Error()
			return
		}

		state.state = stateIdle
		state.err = ""
	}()

	writeJson(w, http.StatusAccepted, currentStatus())
}

func stopMusicHandler(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()

	cancel := state.cancel
	state.cancel = nil
	state.state = stateIdle
	state.err = ""
	state.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	writeJson(w, http.StatusOK, currentStatus())
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	writeJson(w, http.StatusOK, currentStatus())
}

func writeJson(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func currentStatus() statusResponse {
	state.mu.Lock()
	defer state.mu.Unlock()

	return statusResponse{
		StreamUrl: state.streamURL,
		Error:     state.err,
		State:     string(state.state),
	}
}
