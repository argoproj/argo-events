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

package hdfs

import (
	"context"
	"errors"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventsources/common/fsevent"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates hdfs event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.HDFSEventSource)
}

func validate(eventSource *v1alpha1.HDFSEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.Type == "" {
		return errors.New("type is required")
	}
	op := fsevent.NewOp(eventSource.Type)
	if op == 0 {
		return errors.New("type is invalid")
	}
	if eventSource.CheckInterval != "" {
		_, err := time.ParseDuration(eventSource.CheckInterval)
		if err != nil {
			return errors.New("failed to parse interval")
		}
	}
	err := eventSource.WatchPathConfig.Validate()
	if err != nil {
		return err
	}
	if len(eventSource.Addresses) == 0 {
		return errors.New("addresses is required")
	}

	hasKrbCCache := eventSource.KrbCCacheSecret != nil
	hasKrbKeytab := eventSource.KrbKeytabSecret != nil

	if eventSource.HDFSUser == "" && !hasKrbCCache && !hasKrbKeytab {
		return errors.New("either hdfsUser, krbCCacheSecret or krbKeytabSecret is required")
	}
	if hasKrbKeytab && (eventSource.KrbServicePrincipalName == "" || eventSource.KrbConfigConfigMap == nil || eventSource.KrbUsername == "" || eventSource.KrbRealm == "") {
		return errors.New("krbServicePrincipalName, krbConfigConfigMap, krbUsername and krbRealm are required with krbKeytabSecret")
	}
	if hasKrbCCache && (eventSource.KrbServicePrincipalName == "" || eventSource.KrbConfigConfigMap == nil) {
		return errors.New("krbServicePrincipalName and krbConfigConfigMap are required with krbCCacheSecret")
	}
	return err
}
