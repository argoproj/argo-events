# Roadmap

## Development Phase
- E2E testing
- Documentation
- Implement repeatable sensors
- Add NATS Streaming Signal support
- Add SNS & SQS Signal Support


## Design Phase
- Pluggable Calendar definitions that extend a common Calendar interface
- Pass in credentials for connection to various signal sources
- S3 Bucket Notification Handling
- Implement GC for old sensors


## Idea Phase
- Smarter pod queue processing
- Attach PVC to sensor executor pod spec for storing events
