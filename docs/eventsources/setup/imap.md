# IMAP

IMAP event-source monitors email messages in an IMAP mailbox and triggers workloads when new emails arrive.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

        {
            "context": {
              "type": "type_of_event_source",
              "specversion": "cloud_events_version",
              "source": "name_of_the_event_source",
              "id": "unique_event_id",
              "time": "event_time",
              "datacontenttype": "type_of_data",
              "subject": "name_of_the_configuration_within_event_source"
            },
            "data": {
              "subject": "Email subject line",
              "from": "sender@example.com",
              "to": ["recipient1@example.com", "recipient2@example.com"],
              "body": "",
              "headers": {},
              "timestamp": "2025-07-01T13:45:30Z",
              "metadata": {}
            }
        }

## Specification

IMAP event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.IMAPEventSource).

## Setup

1. Create a secret to store IMAP credentials.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/imap-secret.yaml

   **Note**: Update the secret with your actual credentials. For Gmail, you'll need to use an App Password instead of your regular password. See [Gmail App Passwords](https://support.google.com/accounts/answer/185833) for details.

1. Install the event source in the `argo-events` namespace.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/imap.yaml

1. The event-source will start monitoring the INBOX for new emails. It uses IMAP IDLE for real-time notifications when supported, otherwise falls back to polling every 30 seconds. Let's create the sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/imap.yaml

1. Once the sensor pod is in running state, send an email to the configured email address to trigger the workflow.

## Configuration

### Required Fields

- `hostAddress`: IMAP server address with port (e.g., `mail.gmail.com:993`)
- `username`: Reference to Kubernetes secret containing the email username
- `password`: Reference to Kubernetes secret containing the email password

### Optional Fields

- `startTLS`: Enable TLS connection (default: `false`)
- `connectionBackoff`: Backoff configuration for connection retries
- `metadata`: Static metadata to include in all events
- `filter`: Event filtering configuration

### Real-time Notifications

The IMAP event source automatically uses **IMAP IDLE** for real-time email notifications when the server supports it. This provides immediate event triggering when new emails arrive, rather than waiting for the next polling interval.

- **IDLE supported**: Events are triggered instantly when emails arrive
- **IDLE not supported**: Falls back to polling every 30 seconds
- **Automatic detection**: No configuration needed - the event source detects server capabilities

### Example Configuration

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: imap
spec:
  imap:
    example:
      hostAddress: mail.gmail.com:993
      startTLS: true
      username:
        name: imap-secret
        key: username
      password:
        name: imap-secret
        key: password
      metadata:
        source: imap-eventsource
```

## Provider-Specific Configuration

### Gmail
- **Host**: `imap.gmail.com:993`
- **TLS**: Required (`startTLS: true`)
- **Authentication**: Use App Passwords (not regular password)

### Outlook/Hotmail
- **Host**: `outlook.office365.com:993`
- **TLS**: Required (`startTLS: true`)

### Yahoo Mail
- **Host**: `imap.mail.yahoo.com:993`
- **TLS**: Required (`startTLS: true`)

## Security Considerations

1. **Always use TLS**: Set `startTLS: true` for production deployments
2. **Use App Passwords**: For Gmail and other providers that support 2FA
3. **Secret Management**: Store credentials in Kubernetes secrets, never in plain text
4. **Limited Scope**: Consider using dedicated email accounts for automation

## Troubleshoot

1. **Connection Issues**: Verify the hostname and port are correct for your email provider
2. **Authentication Failures**:
    - Ensure you're using the correct credentials
   - For Gmail, use App Passwords instead of regular passwords
   - Check if 2FA is enabled and properly configured
3. **No Events Generated**:
    - Check if new emails are actually arriving in the INBOX
   - Verify the event source pod logs for any errors
   - Ensure the sensor is properly configured to listen for IMAP events

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/) for additional troubleshooting guidance.