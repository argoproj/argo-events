/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package artifact

import (
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

const ArgoEventsEventSourceVersion = "v0.10"

// S3EventSourceExecutor implements Eventing
type S3EventSourceExecutor struct {
	Log *logrus.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

func parseEventSource(config string) (interface{}, error) {
	var a *apicommon.S3Artifact
	err := yaml.Unmarshal([]byte(config), &a)
	if err != nil {
		return nil, err
	}
	return a, err
}
