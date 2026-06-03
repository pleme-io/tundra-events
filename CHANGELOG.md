# Changelog

All notable changes to this project are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-03

### Added
- Initial typed event schema — a generic, transport-independent `Envelope`
  (id/type/source/time/data), a `Payload` marker interface, a built-in
  `SecretEvent` payload for the `secret.*` types, a `Register` seam for
  consumer-defined event types, and `New`/`Validate`/`Bytes`/`Parse`/`Decode`
  with code-carrying errors via `errors-go`. Names no particular project.
