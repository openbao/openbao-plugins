# Changelog

We document all notable changes to this project in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2021-06-24

### Added

* Add `RestartableDescriptor` that can be manually restarted externally to the scheduler lifecycle.

## [0.2.1] - 2021-02-24

### Build

* Update timeutil package to v0.3.0.

## [0.2.0] - 2021-01-06

### Changed

* Updated the recovery descriptor to take a timeutil backoff, which provides considerably more flexibility in the way backoffs and retries are performed.
* Removed dependency on errawr since there's a very minimal chance the APIs in this module will be used for anything outward-facing.

## [0.1.5] - 2020-12-04

### Fixed

* Removed unnecessary `replace` directives in Go module definition.

## [0.1.4] - 2020-12-04

### Fixed

* Removed unnecessary `replace` directives in Go module definition.

## [0.1.3] - 2020-12-04

### Fixed

* Removed unnecessary `replace` directives in Go module definition.

## [0.1.2] - 2020-12-04

### Fixed

* Removed unnecessary `replace` directives in Go module definition.

## [0.1.1] - 2020-12-04

### Fixed

* Removed unnecessary `replace` directives in Go module definition.

## [0.1.0] - 2020-12-04

### Changed

* Renamed project to Leg.

[Unreleased]: https://github.com/puppetlabs/leg/compare/scheduler/v0.3.0...HEAD
[0.3.0]: https://github.com/puppetlabs/leg/compare/scheduler/v0.2.1...scheduler/v0.3.0
[0.2.1]: https://github.com/puppetlabs/leg/compare/scheduler/v0.2.0...scheduler/v0.2.1
[0.2.0]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.5...scheduler/v0.2.0
[0.1.5]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.4...scheduler/v0.1.5
[0.1.4]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.3...scheduler/v0.1.4
[0.1.3]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.2...scheduler/v0.1.3
[0.1.2]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.1...scheduler/v0.1.2
[0.1.1]: https://github.com/puppetlabs/leg/compare/scheduler/v0.1.0...scheduler/v0.1.1
[0.1.0]: https://github.com/puppetlabs/leg/compare/d290e8e835c3fa3ea4e93073bfe19e1958493d47...scheduler/v0.1.0
