# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.0] - 2026-05-01

### Added

- Initial release.
- Generic `Cache[T]` over `string` keys.
- LRU bound from `hashicorp/golang-lru/v2`.
- Per-entry TTL with lazy eviction on `Get`.
- `WithClock` option for deterministic tests.
- `Set`, `Get`, `Remove`, `Len`, `Purge`.

[Unreleased]: https://github.com/ubgo/threadsafecache/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/ubgo/threadsafecache/releases/tag/v0.1.0
