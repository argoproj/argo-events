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

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInlineReader(t *testing.T) {
	myStr := "myStr"
	inlineReader, err := NewInlineReader(&myStr)
	assert.NotNil(t, inlineReader)
	assert.Nil(t, err)
	data, err := inlineReader.Read()
	assert.NotNil(t, data)
	assert.Nil(t, err)
	assert.Equal(t, string(data), myStr)
}
