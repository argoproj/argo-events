# Trigger Argo Workflows from GitHub (Events + Workflows)

This tutorial wires **Argo Events** to **Argo Workflows** so that a GitHub webhook (for example a pull request event) submits a Workflow. It is the end-to-end path teams use for “run a pipeline when code changes.”

If you only need a generic HTTP webhook, start with the [Quick Start](../quick_start.md) instead. Event-source details live in [GitHub EventSource](../eventsources/setup/github.md); trigger details live in [Argo Workflow Trigger](../sensors/triggers/argo-workflow.md).

## What you will install

| Component | Role |
| --- | --- |
| Argo Workflows | Runs `Workflow` objects |
| Argo Events (controller + EventBus) | Receives GitHub webhooks and triggers workflows |
| GitHub EventSource | Registers/handles the GitHub webhook |
| Sensor | Maps a GitHub event to a Workflow create/submit |

```
GitHub webhook ? EventSource Service ? EventBus ? Sensor ? Workflow (Argo Workflows)
```

## Prerequisites

1. A Kubernetes cluster and `kubectl`.
2. [Argo Workflows](https://argoproj.github.io/argo-workflows/) installed so the workflow controller can see the namespace where Sensors create Workflows (cluster-wide install, or `--managed-namespace` including that namespace). Example cluster-scoped install:

        export ARGO_WORKFLOWS_VERSION=3.5.4
        kubectl create namespace argo
        kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v$ARGO_WORKFLOWS_VERSION/install.yaml

3. A GitHub repository where you can add a webhook (or allow Argo Events to create one via API token).

## 1. Install Argo Events

        kubectl create namespace argo-events
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
        # Optional validating admission controller
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install-validating-webhook.yaml

Create an EventBus in the same namespace Sensors/EventSources will use:

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

## 2. RBAC for Sensors and Workflows

The Sensor needs permission to create Workflows. Workflow pods need their own SA as usual:

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/rbac/sensor-rbac.yaml
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/rbac/workflow-rbac.yaml

The example Sensor uses `serviceAccountName: operate-workflow-sa`. Reuse that name or change both the RBAC and the Sensor.

## 3. GitHub credentials

Create a fine-grained or classic personal access token with permission to manage repository hooks (`repo` or hook-related scopes). Base64-encode the token and an optional webhook secret:

        echo -n "YOUR_GITHUB_TOKEN" | base64
        echo -n "YOUR_WEBHOOK_SECRET" | base64

        kubectl apply -n argo-events -f - <<'EOF'
        apiVersion: v1
        kind: Secret
        metadata:
          name: github-access
        type: Opaque
        data:
          token: YOUR_BASE64_TOKEN
          secret: YOUR_BASE64_WEBHOOK_SECRET
        EOF

## 4. Deploy the GitHub EventSource

Start from the upstream example and set your repository and webhook URL:

        curl -sL https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/github.yaml -o github-eventsource.yaml

Edit at least:

* `spec.github.example.repositories` — owner/name of your repo
* `spec.github.example.webhook.url` — **publicly reachable** URL of the EventSource service (Ingress, LoadBalancer, or a tunnel for local demos)

The EventSource Service name is `<event-source-name>-eventsource-svc` (for the example, `github-eventsource-svc`) on the webhook port from the manifest.

Apply it:

        kubectl apply -n argo-events -f github-eventsource.yaml

Expose the service (choose one):

* **Ingress / LoadBalancer** for real GitHub webhooks.
* **Local debug:** `kubectl -n argo-events port-forward svc/github-eventsource-svc 12000:12000` plus a tunnel (ngrok, etc.) if GitHub must reach your laptop.

Confirm GitHub shows the webhook under repository **Settings ? Webhooks**, or check the EventSource pod logs.

## 5. Deploy the Sensor (trigger a Workflow)

The example Sensor listens for pull request events and creates a Workflow, parameterizing title, number, and SHA from the payload:

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/github.yaml

Notable parts of that manifest:

* `dependencies` — `eventSourceName: github`, `eventName: example`, with filters on `X-Github-Event` and PR actions.
* `triggers` — a Kubernetes trigger that creates `kind: Workflow` (`apiVersion: argoproj.io/v1alpha1`), or you can use the dedicated `argoWorkflow` trigger (submit/resubmit/etc.) as in [Argo Workflow Trigger](../sensors/triggers/argo-workflow.md).
* `parameters` — copies JSON fields from the GitHub body into Workflow arguments.

### Trigger an existing WorkflowTemplate

If your pipeline is already packaged as a `WorkflowTemplate`, submit a thin Workflow that only references it:

```yaml
triggers:
  - template:
      name: github-workflow-trigger
      argoWorkflow:
        operation: submit
        source:
          resource:
            apiVersion: argoproj.io/v1alpha1
            kind: Workflow
            metadata:
              generateName: github-pipeline-
            spec:
              workflowTemplateRef:
                name: your-workflow-template
```

See [examples/sensors/sensor-to-existing-workflow-template.yaml](https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/sensor-to-existing-workflow-template.yaml).

## 6. Test the integration

1. Open or synchronize a pull request against the branch filtered in the Sensor (the example uses `master` as `body.pull_request.base.ref` — change the filter to `main` if needed).
2. List workflows in the Sensor namespace:

        kubectl -n argo-events get workflows
        # or: argo list -n argo-events

3. Inspect the Sensor and EventSource if nothing fires:

        kubectl -n argo-events get eventsource,sensor
        kubectl -n argo-events logs -l eventsource-name=github
        kubectl -n argo-events logs -l sensor-name=github

## 7. Optional: install with Argo CD

Treat Argo Events and Argo Workflows as two Argo CD Applications (or one app-of-apps):

1. App **argo-workflows** — manifests or Helm chart from the Workflows release into namespace `argo`.
2. App **argo-events** — `manifests/install.yaml` plus your EventBus, EventSource, Sensor, and RBAC into `argo-events`.
3. Keep secrets (`github-access`) out of git; use a Secret manager or Sealed Secrets.

Webhook URLs in the EventSource must match the Ingress/Service your cluster exposes after sync.

## Related docs

* [Quick Start](../quick_start.md) — webhook ? Workflow on your laptop
* [GitHub EventSource setup](../eventsources/setup/github.md)
* [Argo Workflow trigger](../sensors/triggers/argo-workflow.md)
* [Parameterization](02-parameterization.md) — pass commit SHA, branch, and PR metadata into workflows
* [Installation](../installation.md)
