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

package amqp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYamlTable(t *testing.T) {
	table, err := parseYamlTable("")
	assert.Nil(t, err)
	assert.Nil(t, table)
	table, err = parseYamlTable(`:noKey`)
	assert.NotNil(t, err)
	assert.Nil(t, table)
	table, err = parseYamlTable("x-queue-type: quorum")
	assert.Nil(t, err)
	assert.NotNil(t, table)
	assert.True(t, len(table) == 1)
	table, err = parseYamlTable("key-one: thing1\nkey-two: thing2")
	assert.Nil(t, err)
	assert.NotNil(t, table)
	assert.True(t, len(table) == 2)
	assert.Equal(t, "thing1", table["key-one"].(string))
	assert.Equal(t, "thing2", table["key-two"].(string))
}
