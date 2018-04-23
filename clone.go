// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worksheets

import (
	"fmt"

	"github.com/satori/go.uuid"
)

// Clone duplicates this worksheet, and all worksheets it points to, in order
// to create a deep-copy.
func (ws *Worksheet) Clone() *Worksheet {
	c := &cloner{
		mapping: make(map[string]string),
		clones:  make(map[string]*Worksheet),
	}

	return c.cloneWs(ws)
}

type cloner struct {
	// mapping maps original ws ids, to dupped ws ids
	mapping map[string]string

	// clones records all dupped worksheets by their ids
	clones map[string]*Worksheet
}

func (c *cloner) clone(parent *Worksheet, index int, value Value) Value {
	switch v := value.(type) {
	case *Worksheet:
		child := c.cloneWs(v)
		child.parents.addParentViaFieldIndex(parent, index)
		return child
	case *Slice:
		dupSlice := newSlice(v.typ)
		for _, element := range v.Elements() {
			var err error
			dupElement := c.clone(parent, index, element)
			dupSlice, err = dupSlice.doAppend(dupElement)
			if err != nil {
				panic(fmt.Sprintf("unexpected %s", err))
			}
		}
		return dupSlice
	default:
		return value
	}
}

func (c *cloner) cloneWs(ws *Worksheet) *Worksheet {
	if _, ok := c.mapping[ws.Id()]; ok {
		return c.clones[c.mapping[ws.Id()]]
	}

	// When duplicating a worksheet, we change the underlying data structures
	// directly to make an exact copy of the values, rather than go through
	// constrained fields, computed fields, etc. This guarantees that the
	// copy is done at the value level, and yields the same data, even in the
	// case where definitions for various fields have changed since the creation
	// of ws.
	// The duplicated worksheet is a fresh new instance, with its own id, and
	// its version set at 1.

	dup := ws.def.newUninitializedWorksheet()
	dup.data[indexId] = NewText(uuid.Must(uuid.NewV4()).String())
	dup.data[indexVersion] = NewNumberFromInt(1)
	c.mapping[ws.Id()] = dup.Id()
	c.clones[dup.Id()] = dup

	for index, value := range ws.data {
		if 0 < index {
			dup.data[index] = c.clone(dup, index, value)
		}
	}

	return dup
}
