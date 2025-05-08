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

import (
	"github.com/spf13/pflag"
)

const (
	fromFlagPrefix = "from"
	fromNotePrefix = "source"
)

type RemoteFrom struct {
	Remote
}

// TODO remove getters probably
func (rf *RemoteFrom) getFlagPrefix() (flagPrefix string) {
	return fromFlagPrefix + "-"
}

func (rf *RemoteFrom) getNotePrefix() (notePrefix string) {
	return fromNotePrefix + " "
}

func (rf *RemoteFrom) UpdateFlags(fs *pflag.FlagSet) {
	rf.ApplyFlagsWithPrefix(fs, rf.getFlagPrefix(), rf.getNotePrefix())
}
