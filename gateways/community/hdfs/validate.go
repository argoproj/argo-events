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
	"path"
	"time"

	"github.com/argoproj/argo-events/gateways/common/naivewatcher"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *EventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es.Data, parseEventSource, validateGatewayConfig)
}

func validateGatewayConfig(config interface{}) error {
	gwc := config.(*GatewayConfig)
	if gwc == nil {
		return gwcommon.ErrNilEventSource
	}
	if gwc.Directory == "" {
		return errors.New("directory is required")
	}
	if !path.IsAbs(gwc.Directory) {
		return errors.New("directory must be an absolute file path")
	}
	if gwc.Path == "" {
		return errors.New("path is required")
	}
	if path.IsAbs(gwc.Path) {
		return errors.New("path must be a relative file path")
	}
	if gwc.Type == "" {
		return errors.New("type is required")
	}
	op := naivewatcher.NewOp(gwc.Type)
	if op == 0 {
		return errors.New("type is invalid")
	}
	if gwc.CheckInterval != "" {
		_, err := time.ParseDuration(gwc.CheckInterval)
		if err != nil {
			return errors.New("failed to parse interval")
		}
	}

	err := validateHDFSConfig(gwc)

	return err
}

func validateHDFSConfig(gwc *GatewayConfig) error {
	if len(gwc.Addresses) == 0 {
		return errors.New("addresses is required")
	}

	hasKrbCCache := gwc.KrbCCacheSecret != nil
	hasKrbKeytab := gwc.KrbKeytabSecret != nil

	if gwc.HDFSUser == "" && !hasKrbCCache && !hasKrbKeytab {
		return errors.New("either hdfsUser, krbCCacheSecret or krbKeytabSecret is required")
	}
	if hasKrbKeytab && (gwc.KrbServicePrincipalName == "" || gwc.KrbConfigConfigMap == nil || gwc.KrbUsername == "" || gwc.KrbRealm == "") {
		return errors.New("krbServicePrincipalName, krbConfigConfigMap, krbUsername and krbRealm are required with krbKeytabSecret")
	}
	if hasKrbCCache && (gwc.KrbServicePrincipalName == "" || gwc.KrbConfigConfigMap == nil) {
		return errors.New("krbServicePrincipalName and krbConfigConfigMap are required with krbCCacheSecret")
	}

	return nil
}
