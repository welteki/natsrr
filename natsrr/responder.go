package natsrr

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/nats-io/nats.go"
)

// A Responder interface is used by an handler to
// construct a nats response message.
type Responder interface {
	// SetStatus sends a response message with the provided status code
	// in the headers.
	SetStatus(statusCode int)

	// SetDescription sends a response message with the provided status
	// description in the headers.
	SetDescription(description string)

	// Header returns the header map that will be sent by
	// reply message.
	Header() nats.Header

	// Respond with the data as part of the response message
	Respond(data []byte) error
}

type responder struct {
	msg  *nats.Msg
	resp *nats.Msg
}

func newResponder(msg *nats.Msg) *responder {
	response := &responder{
		msg:  msg,
		resp: nats.NewMsg(msg.Reply),
	}

	response.resp.Header.Set("Status", strconv.Itoa(http.StatusOK))

	return response
}

func (r *responder) SetStatus(statusCode int) {
	r.resp.Header.Set("Status", strconv.Itoa(statusCode))
}

func (r *responder) SetDescription(description string) {
	r.resp.Header.Set("Description", description)
}

func (r *responder) Respond(data []byte) error {
	r.resp.Data = data
	return r.msg.RespondMsg(r.resp)
}

func (r *responder) Header() nats.Header {
	return r.resp.Header
}

// Error responds to the message with the specified error description and status.
func Error(r Responder, err string, code int) {
	r.SetStatus(code)
	r.SetDescription(err)
	r.Respond(nil)
}

// The HandlerFunc type allow the use of
// the responder to handle nats messages.
type HandlerFunc func(Responder, *nats.Msg)

// HandleMsg calls f(responder, msg).
func (f HandlerFunc) HandleMsg(msg *nats.Msg) {
	f(newResponder(msg), msg)
}

// SubjectMux is a nats message multiplexer.
// It matches the subject of each incoming subject against a list of registered
// patterns and calls the handler for the pattern that
// matches the subject.
type SubjectMux struct {
	mu sync.RWMutex
	m  map[string]muxEntry
}

type muxEntry struct {
	h    HandlerFunc
	subj string
}

// NewSubjectMux allocates and returns a new SubjectMux.
func NewSubjectMux() *SubjectMux { return new(SubjectMux) }

// MsgHandler returns the nats MsgHandler that dispatches the message
// to the handler that matches the subject.
//
// If there is no registered handler that applies to the message,
// the message is dispatched to the NotFound handler.
func (mux *SubjectMux) MsgHandler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		mux.handler(msg.Subject).HandleMsg(msg)
	}
}

// handler return the handler to use for the given message.
func (mux *SubjectMux) handler(subj string) HandlerFunc {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	v, ok := mux.m[subj]
	if ok {
		return v.h
	}

	return NotFound
}

// HandleFunc registers the handler function for the given subject.
func (mux *SubjectMux) HandleFunc(subj string, handler func(Responder, *nats.Msg)) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if subj == "" {
		panic("natsrr: invalid pattern")
	}
	if handler == nil {
		panic("natsrr: nil handler")
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	e := muxEntry{h: handler, subj: subj}
	mux.m[subj] = e
}

// NotFound responds to the message with an 404 status, no messages error.
func NotFound(r Responder, msg *nats.Msg) { Error(r, "No messages", http.StatusNotFound) }
