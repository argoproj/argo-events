package driver

import nats "github.com/nats-io/nats.go"

type JjetstreamConnection struct {
	natsConn  *nats.Conn
	jsContext nats.JetStreamContext

	natsConnected bool
}

func (jsc *JjetstreamConnection) Close() error {

	if jsc.natsConn != nil && jsc.natsConn.IsConnected() {
		jsc.natsConn.Close()
	}
	return nil
}

func (jsc *JjetstreamConnection) IsClosed() bool {
	if jsc.natsConn == nil || !jsc.natsConnected || jsc.natsConn.IsClosed() {
		return true
	}
	return false
}

func (jsc *JjetstreamConnection) Publish(subject string, data []byte) error {
	// todo: On the publishing side you can avoid duplicate message ingestion using the Message Deduplication feature.
	_, err := jsc.jsContext.Publish(subject, data)
	return err
}
