// Package events is the fleet's one typed event schema — a generic Envelope
// wrapping a typed payload — so no event-consuming tool hand-rolls a bespoke
// JSON event model with ad-hoc reconnect/filter logic. The Envelope is the
// transport-independent shape; payloads are typed and registered by event type.
//
// It is a pure, generic event-schema primitive: it models any event source
// (an Event Center, a message bus, a webhook) and never names or imports any
// particular project.
package events

import (
	"encoding/json"
	"time"

	errs "github.com/pleme-io/errors-go"
)

// Type identifies the kind of event an Envelope carries; it selects the payload
// shape. New types are additive.
type Type string

// Built-in event types. Consumers may register their own via Register.
const (
	// SecretCreated signals a new secret was created at Payload.Path.
	SecretCreated Type = "secret.created"
	// SecretUpdated signals a secret's value rotated/changed at Payload.Path.
	SecretUpdated Type = "secret.updated"
	// SecretDeleted signals a secret was removed at Payload.Path.
	SecretDeleted Type = "secret.deleted"
)

// Payload is the marker interface every typed event payload satisfies. A
// payload reports the Type it belongs to so an Envelope can be built and
// validated from the payload alone.
type Payload interface {
	// EventType returns the Type this payload is the body of.
	EventType() Type
}

// SecretEvent is the typed payload for the secret.* event types — the generic
// shape a secret-lifecycle source emits. It carries no vendor-specific fields.
type SecretEvent struct {
	// Kind is the specific secret event type (created/updated/deleted).
	Kind Type `json:"kind"`
	// Path is the logical secret path the event concerns.
	Path string `json:"path"`
	// Version is the new version after the change, if known.
	Version int `json:"version,omitempty"`
	// Tags are arbitrary labels attached to the secret.
	Tags []string `json:"tags,omitempty"`
}

// EventType returns the secret event's Kind, satisfying Payload.
func (e SecretEvent) EventType() Type { return e.Kind }

// Envelope is the generic, transport-independent event wrapper. It carries
// metadata (id, type, source, time) plus the typed payload as raw JSON so it can
// be decoded into the registered concrete payload type.
type Envelope struct {
	// ID is a unique event id (idempotency key for at-least-once delivery).
	ID string `json:"id"`
	// Type selects the payload shape.
	Type Type `json:"type"`
	// Source names the emitter (a URL/hostname/logical name).
	Source string `json:"source"`
	// Time is when the event occurred (UTC).
	Time time.Time `json:"time"`
	// Data is the typed payload, carried as raw JSON.
	Data json.RawMessage `json:"data"`
}

// Option configures an Envelope at construction.
type Option func(*Envelope)

// WithID sets the event id.
func WithID(id string) Option { return func(e *Envelope) { e.ID = id } }

// WithSource sets the emitter source.
func WithSource(src string) Option { return func(e *Envelope) { e.Source = src } }

// WithTime sets the event time (defaults to time.Now().UTC() in New).
func WithTime(t time.Time) Option { return func(e *Envelope) { e.Time = t } }

// New builds an Envelope around a typed payload. The Envelope's Type is derived
// from the payload (payload.EventType), and Data is the payload marshalled to
// JSON. A nil payload, or a JSON-unmarshalable payload, yields a typed error.
func New(payload Payload, opts ...Option) (*Envelope, error) {
	if payload == nil {
		return nil, errs.New("events: nil payload", errs.WithCode("event_nil_payload"))
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.Wrap(err, "events: payload is not JSON-marshalable", errs.WithCode("event_marshal"))
	}
	e := &Envelope{
		Type: payload.EventType(),
		Time: time.Now().UTC(),
		Data: raw,
	}
	for _, o := range opts {
		o(e)
	}
	return e, nil
}

// Validate reports the first structural problem with the envelope as a typed,
// code-carrying error. A valid envelope has an id and a registered type.
func (e *Envelope) Validate() error {
	if e.ID == "" {
		return errs.New("events: envelope has no id", errs.WithCode("event_no_id"))
	}
	if e.Type == "" {
		return errs.New("events: envelope has no type", errs.WithCode("event_no_type"))
	}
	if _, ok := registry[e.Type]; !ok {
		return errs.New("events: unregistered event type "+string(e.Type), errs.WithCode("event_unregistered"))
	}
	return nil
}

// MarshalJSON / UnmarshalJSON use the default struct tags — Envelope is a plain
// JSON document, so the transport (AMQP, HTTP, NATS) just carries bytes.
func (e *Envelope) Bytes() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, errs.Wrap(err, "events: marshal envelope", errs.WithCode("event_marshal"))
	}
	return b, nil
}

// Parse decodes raw bytes into an Envelope (without decoding the payload). Use
// Decode to obtain the typed payload.
func Parse(b []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, errs.Wrap(err, "events: parse envelope", errs.WithCode("event_parse"))
	}
	return &e, nil
}

// registry maps an event Type to a constructor for its zero payload, so Decode
// can produce a concrete typed value from an Envelope.
var registry = map[Type]func() Payload{}

// Register associates an event Type with a factory for its payload type. It is
// the seam consumers use to add their own typed events. Built-in secret.* types
// are registered in init.
func Register(t Type, factory func() Payload) {
	registry[t] = factory
}

func init() {
	Register(SecretCreated, func() Payload { return &SecretEvent{Kind: SecretCreated} })
	Register(SecretUpdated, func() Payload { return &SecretEvent{Kind: SecretUpdated} })
	Register(SecretDeleted, func() Payload { return &SecretEvent{Kind: SecretDeleted} })
}

// Decode validates the envelope and decodes its Data into the registered
// concrete payload type for e.Type.
func Decode(e *Envelope) (Payload, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	p := registry[e.Type]()
	if err := json.Unmarshal(e.Data, p); err != nil {
		return nil, errs.Wrap(err, "events: decode payload for "+string(e.Type), errs.WithCode("event_decode"))
	}
	return p, nil
}
