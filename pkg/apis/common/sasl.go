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

import corev1 "k8s.io/api/core/v1"

// SASLConfig refers to SASL configuration for a client
type SASLConfig struct {
	// SASLMechanism is the name of the enabled SASL mechanism.
	// Possible values: OAUTHBEARER, PLAIN (defaults to PLAIN).
	// +optional
	Mechanism string `json:"mechanism,omitempty" protobuf:"bytes,1,opt,name=mechanism"`
	// User is the authentication identity (authcid) to present for
	// SASL/PLAIN or SASL/SCRAM authentication
	UserSecret *corev1.SecretKeySelector `json:"userSecret,omitempty" protobuf:"bytes,2,opt,name=userSecret"`
	// Password for SASL/PLAIN authentication
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty" protobuf:"bytes,3,opt,name=passwordSecret"`
}

func (s SASLConfig) GetMechanism() string {
	switch s.Mechanism {
	case "OAUTHBEARER", "SCRAM-SHA-256", "SCRAM-SHA-512", "GSSAPI":
		return s.Mechanism
	default:
		// default to PLAINTEXT mechanism
		return "PLAIN"
	}
}
