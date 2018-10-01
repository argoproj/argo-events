# Roadmap

## Development Phase
- E2E testing
- Documentation
- Add NATS Streaming as internal pub-sub system
- Add SNS & SQS gateway Support

## Idea Phase
- Smarter pod queue processing
- Attach PVC to sensor executor pod spec for storing events
- Use [hashicorp/go-getter](https://github.com/hashicorp/go-getter) for the `ArtifactReader` interface.
- Add [Upspin](https://upspin.io/) as an `Artifact` file source.