/*
Copyright The ORAS Authors.
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

package model

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// push contains metadata formatted by oras push.
type push struct {
	Descriptor
	ReferenceAsTags []string `json:"referenceAsTags"`
}

// NewPush returns a metadata getter for push command.
func NewPush(desc ocispec.Descriptor, path string, tags []string) any {
	var refAsTags []string
	for _, tag := range tags {
		refAsTags = append(refAsTags, path+":"+tag)
	}
	return push{
		Descriptor:      FromDescriptor(path, desc),
		ReferenceAsTags: refAsTags,
	}
}
