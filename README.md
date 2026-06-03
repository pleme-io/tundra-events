# tundra-events

A typed event schema — a generic, transport-independent `Envelope` wrapping
typed payloads.

## What

Model an event as an `Envelope` (id, type, source, time, raw-JSON data) carrying
a typed `Payload`. Build one with `New(payload, opts…)`, serialize with
`Bytes`, `Parse` raw bytes back, and `Decode` into the concrete payload type
registered for its `Type`. A built-in `SecretEvent` covers the `secret.*` types;
consumers register their own via `Register`. It is a pure, generic event-schema
primitive: it models any source (Event Center, message bus, webhook) and names
no particular project.

## Why

Every event-consuming service otherwise hand-rolls its own JSON event model with
bespoke reconnect/filter logic and no shared shape. tundra-events is the one
shared envelope so producers and consumers agree on the wire format, and the
`id` field doubles as the idempotency key for at-least-once delivery.

## Install

```
go get github.com/pleme-io/tundra-events
```

Nix (via substrate):

```nix
outputs = { self, nixpkgs, substrate, ... }:
  (import substrate.goLibraryFlakeBuilder { inherit nixpkgs; }) {
    name = "tundra-events"; version = "0.1.0"; src = self;
  };
```

## Usage

Built on: [errors-go] (typed, code-carrying errors).

```go
env, err := events.New(
    events.SecretEvent{Kind: events.SecretUpdated, Path: "/db/prod", Version: 3},
    events.WithID("evt-42"), events.WithSource("event-center"),
)
if err != nil { return errs.Exit(err) }

b, _ := env.Bytes()         // publish over AMQP/HTTP/NATS

parsed, _ := events.Parse(b) // consumer side
payload, err := events.Decode(parsed)
se := payload.(*events.SecretEvent)
```

Register a custom event type:

```go
events.Register("custom.thing", func() events.Payload { return &MyPayload{} })
```

## Configuration

None — it is a pure library. A consumer that loads broker/topic settings from a
configured source uses `shikumi-go` for that config; tundra-events only models
the event itself.

## Release

Pull-model (Go modules): an annotated `vX.Y.Z` tag is the release;
`proxy.golang.org` + pkg.go.dev index it. See the GSDS module delivery FSM.

[errors-go]: https://github.com/pleme-io/errors-go
