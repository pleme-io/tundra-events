package events

import (
	"testing"
	"time"

	errs "github.com/pleme-io/errors-go"
)

func TestNewAndRoundTrip(t *testing.T) {
	payload := SecretEvent{Kind: SecretUpdated, Path: "/db/prod", Version: 3, Tags: []string{"a", "b"}}
	env, err := New(payload, WithID("evt-1"), WithSource("event-center"))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if env.Type != SecretUpdated {
		t.Fatalf("type = %q, want %q", env.Type, SecretUpdated)
	}
	if env.Time.IsZero() {
		t.Fatal("time not set")
	}

	b, err := env.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	parsed, err := Parse(b)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := Decode(parsed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	se, ok := got.(*SecretEvent)
	if !ok {
		t.Fatalf("decoded type %T, want *SecretEvent", got)
	}
	if se.Path != "/db/prod" || se.Version != 3 || len(se.Tags) != 2 {
		t.Fatalf("roundtrip payload wrong: %+v", se)
	}
	if se.EventType() != SecretUpdated {
		t.Fatalf("payload event type = %q", se.EventType())
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := New(nil); err == nil || errs.CodeOf(err) != "event_nil_payload" {
		t.Fatalf("nil payload: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		env  *Envelope
		code string // "" = ok
	}{
		{
			name: "ok",
			env:  &Envelope{ID: "x", Type: SecretCreated, Time: time.Now()},
			code: "",
		},
		{
			name: "no id",
			env:  &Envelope{Type: SecretCreated},
			code: "event_no_id",
		},
		{
			name: "no type",
			env:  &Envelope{ID: "x"},
			code: "event_no_type",
		},
		{
			name: "unregistered",
			env:  &Envelope{ID: "x", Type: "nope.unknown"},
			code: "event_unregistered",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if tt.code == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want code %q, got nil", tt.code)
			}
			if errs.CodeOf(err) != tt.code {
				t.Fatalf("code = %q, want %q", errs.CodeOf(err), tt.code)
			}
		})
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	_, err := Parse([]byte("{not json"))
	if err == nil {
		t.Fatal("want parse error")
	}
	if errs.CodeOf(err) != "event_parse" {
		t.Fatalf("code = %q, want event_parse", errs.CodeOf(err))
	}
}

// customPayload exercises the Register seam for consumer-defined event types.
type customPayload struct {
	Kind  Type   `json:"kind"`
	Field string `json:"field"`
}

func (c customPayload) EventType() Type { return c.Kind }

func TestRegisterCustomType(t *testing.T) {
	const ct Type = "custom.thing"
	Register(ct, func() Payload { return &customPayload{Kind: ct} })

	env, err := New(customPayload{Kind: ct, Field: "hello"}, WithID("c-1"))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	got, err := Decode(env)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	cp, ok := got.(*customPayload)
	if !ok || cp.Field != "hello" {
		t.Fatalf("custom decode wrong: %#v", got)
	}
}

func TestDecodeValidatesFirst(t *testing.T) {
	_, err := Decode(&Envelope{Type: SecretCreated}) // missing id
	if err == nil || errs.CodeOf(err) != "event_no_id" {
		t.Fatalf("want event_no_id, got %v", err)
	}
}

// Compile-time proof SecretEvent satisfies Payload.
var _ Payload = SecretEvent{}
