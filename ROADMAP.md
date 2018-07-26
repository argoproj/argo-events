# Roadmap

## Development Phase
- Simplify logging
- E2E testing
- Documentation
- Add NATS Streaming Signal support
- Add SNS & SQS Signal Support


## Design Phase
- Pass in credentials for connection to various signal sources
- Implement GC for old sensors
- Implement repeatable sensors

## Idea Phase
- Smarter pod queue processing
- Attach PVC to sensor executor pod spec for storing events
- Use [hashicorp/go-getter](https://github.com/hashicorp/go-getter) for the `ArtifactReader` interface.
- Add [Upspin](https://upspin.io/) as an `Artifact` file source.