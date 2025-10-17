# Stress Testing

Argo Events provides a set of simple tools to do stress testing:

- An [events generator](generator/README.md) for Event Sources
- An application to create testing EventSources and Sensors, or hook into your
  existing ones, grab data and provide reports

## Before You Start

- Set up Prometheus metrics monitoring.

  Follow the [instructions](https://argoproj.github.io/argo-events/metrics/) to
  set up Prometheus to grab metrics, also make sure the basic Kubernetes metrics
  like Pod CPU/memory usage are captured. Display the metrics using tools like
  [Grafana](https://grafana.com/).

- Create the EventSource and Sensor for testing.

  You can use the tool below to create the EventSource and Sensor, or use your
  existing ones for testing. If you want to run the testing against a webhook
  typed EventSource (e.g. `webhook`, `sns`, etc), you need to set up the ingress
  for it beforehand.

```shell
  # Make sure you have sourced a KUBECONFIG file
  go run ./test/stress/main.go --help
```

## Steps

- Use the [Generator](generator/README.md) or any other generators your have to
  generate messages for your EventSource.

- Explore the metrics in Grafana, watch the changes of
  [golden metrics](https://argoproj.github.io/argo-events/metrics/#golden-signals)
  such as Errors, Latency and Saturation.

## About This Tool

This tool can be used to create a set of EventBus, EventSource and Sensor for
testing in your cluster.

For example, command below creates a `webhook` EventSource and a Sensor with
`log` trigger, and it captures the data and provides a simple report afterwards.
It will exit in 5 minutes and clean up the created resources.

```shell
go run ./test/stress/main.go --eb-type jetstream --es-type webhook --trigger-type log --hard-timeout 5m
```

The spec of `webhook` EventSource is located in
[testdata/eventsources](testdata/eventsources), and the Sensor for `log` trigger
is in [testdata/sensors](testdata/sensors), make sure you update the specs with
desired CPU/memory you want to use before running the command.

The report looks like below:

```text
++++++++++++++++++++++++ Events Summary +++++++++++++++++++++++
Event Name                    : test
Total processed events        : 243
Events sent successful        : 243
Events sent failed            : 0
First event sent at           : 2021-03-19 09:42:03.895959 -0700 PDT m=+101.008343356
Last event sent at            : 2021-03-19 09:42:08.251331 -0700 PDT m=+105.363714318
Total time taken              : 4.355370962s
--

+++++++++++++++++++++++ Actions Summary +++++++++++++++++++++++
Trigger Name                  : log-trigger
Total triggered actions       : 243
Action triggered successfully : 243
Action triggered failed       : 0
First action triggered at     : 2021-03-19 09:42:03.907679 -0700 PDT m=+101.020062977
Last action triggered at      : 2021-03-19 09:42:08.253928 -0700 PDT m=+105.366311326
Total time taken              : 4.346248349s
--
```

It could also hook into your existing EventSources and Sensors to give a simple
report.

```shell
go run ./test/stress/main.go --eb-type jetstream --es-name my-sqs-es --sensor-name my-sensor --hard-timeout 5m
```
