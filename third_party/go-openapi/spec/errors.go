// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spec

import "errors"

// Error codes
var (
	// ErrUnknownTypeForReference indicates that a resolved reference was found in an unsupported container type
	ErrUnknownTypeForReference = errors.New("unknown type for the resolved reference")

	// ErrResolveRefNeedsAPointer indicates that a $ref target must be a valid JSON pointer
	ErrResolveRefNeedsAPointer = errors.New("resolve ref: target needs to be a pointer")

	// ErrDerefUnsupportedType indicates that a resolved reference was found in an unsupported container type.
	// At the moment, $ref are supported only inside: schemas, parameters, responses, path items
	ErrDerefUnsupportedType = errors.New("deref: unsupported type")

	// ErrExpandUnsupportedType indicates that $ref expansion is attempted on some invalid type
	ErrExpandUnsupportedType = errors.New("expand: unsupported type. Input should be of type *Parameter or *Response")
)
