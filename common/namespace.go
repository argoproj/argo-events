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

package common

import (
	"errors"
	"io/ioutil"
	"os"
)

const namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

var (
	// DefaultSensorControllerNamespace is the default namespace where the sensor controller is installed
	DefaultSensorControllerNamespace = "default"

	// ErrReadNamespace occurs when the namespace cannot be read from a Kubernetes pod's service account token
	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
)

func init() {
	RefreshNamespace()
}

// detectNamespace attemps to read the namespace from the mounted service account token
// Note that this will return an error if running outside a Kubernetes pod
func detectNamespace() (string, error) {
	// Make sure it's a file and we can read it
	if s, e := os.Stat(namespacePath); e != nil {
		return "", e
	} else if s.IsDir() {
		return "", ErrReadNamespace
	}

	// Read the file, and cast to a string
	ns, e := ioutil.ReadFile(namespacePath)
	return string(ns), e
}

// RefreshNamespace performs waterfall logic for choosing a "default" namespace
// this function is run as part of an init() function
func RefreshNamespace() {
	// 1 - env variable
	nm, ok := os.LookupEnv(EnvVarNamespace)
	if ok {
		DefaultSensorControllerNamespace = nm
		return
	}

	// 2 - pod service account token
	nm, err := detectNamespace()
	if err == nil {
		DefaultSensorControllerNamespace = nm
	}

	// 3 - use the DefaultSensorControllerNamespace
	return
}
