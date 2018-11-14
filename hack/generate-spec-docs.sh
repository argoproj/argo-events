#!/bin/bash

set -x
set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname "$0")/.. ; pwd)

swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/sensor.json -o ${PROJECT_ROOT}/hack/spec-docs/sensor.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/gateway.json -o ${PROJECT_ROOT}/hack/spec-docs/gateway.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/artifact.json -o ${PROJECT_ROOT}/hack/spec-docs/artifact.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/calendar.json -o ${PROJECT_ROOT}/hack/spec-docs/calendar.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/webhook.json -o ${PROJECT_ROOT}/hack/spec-docs/webhook.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/resource.json -o ${PROJECT_ROOT}/hack/spec-docs/resource.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/file.json -o ${PROJECT_ROOT}/hack/spec-docs/file.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/amqp.json -o ${PROJECT_ROOT}/hack/spec-docs/amqp.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/mqtt.json -o ${PROJECT_ROOT}/hack/spec-docs/mqtt.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/nats.json -o ${PROJECT_ROOT}/hack/spec-docs/nats.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/kafka.json -o ${PROJECT_ROOT}/hack/spec-docs/kafka.md
swagger2markdown -i ${PROJECT_ROOT}/hack/swagger-spec/storagegrid.json -o ${PROJECT_ROOT}/hack/spec-docs/storagegrid.md
