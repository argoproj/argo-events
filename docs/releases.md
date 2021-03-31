# Releases

[Latest releases](https://github.com/argoproj/argo-events/releases)

## Supported Versions

Versions are expressed as x.y.z, where x is the major version, y is the minor
version, and z is the patch version, following Semantic Versioning terminology.

We maintain release branches for the most recent two minor releases.

Fixes may be backported to release branches, depending on severity, risk, and,
feasibility.

If a release contains breaking changes, or CVE fixes, this will documented in
the release notes.

## Supported Version Skew

Image versions of `eventsource`, `sensor`, `eventbus-controller`,
`eventsource-controller`, `sensor-controller` and `events-webhook` should be the
same.

## Release Cycle

For **unstable**, we build and tag `latest` images for every commit to master.

New major versions are released roughly every 6 months. Release candidates for
each major release are typically available for 6 weeks before the release
becomes generally available.

Otherwise, we typically release once a month:

- Patch fixes for the current stable version. These are tagged `stable`.
- The next release candidate, if we are currently in a release-cycle.
