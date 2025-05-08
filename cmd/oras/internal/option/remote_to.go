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

package option

import "github.com/spf13/pflag"

const (
	toFlagPrefix = "to"
	toNotePrefix = "destination"
)

type RemoteTo struct {
	Remote
}

func (rt *RemoteTo) getFlagPrefix() (flagPrefix string) {
	return toFlagPrefix + "-"
}

func (rt *RemoteTo) getNotePrefix() (notePrefix string) {
	return toNotePrefix + " "
}

func (rt *RemoteTo) UpdateFlags(fs *pflag.FlagSet) {
	rt.ApplyFlagsWithPrefix(fs, rt.getFlagPrefix(), rt.getNotePrefix())
}
