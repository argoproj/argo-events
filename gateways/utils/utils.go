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

package utils

import (
	"encoding/json"
	"github.com/argoproj/argo-events/controllers/gateway/transform"
	zlog "github.com/rs/zerolog"
	hs "github.com/mitchellh/hashstructure"
	"os"
)

// TransformerPayload creates a new payload from input data and adds source information
func TransformerPayload(b []byte, source string) ([]byte, error) {
	tp := &transform.TransformerPayload{
		Src:     source,
		Payload: b,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// Logger returns a JSON output logger.
func Logger(name string) zlog.Logger {
	return zlog.New(os.Stdout).With().Str("name", name).Logger()
}

func Hasher(key string, value string) (uint64, error) {
	return hs.Hash(key+value, &hs.HashOptions{})
}