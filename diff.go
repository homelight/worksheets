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
	"bytes"
	"reflect"
)

func (value *Undefined) diffCompare(that Value) bool {
	return value.Equal(that)
}

func (value *Number) diffCompare(that Value) bool {
	return value.Equal(that)
}

func (value *Bool) diffCompare(that Value) bool {
	return value.Equal(that)
}

func (value *Text) diffCompare(that Value) bool {
	return value.Equal(that)
}

func (value *Slice) diffCompare(that Value) bool {
	return value.Equal(that)
}

func (ws *Worksheet) diffCompare(other Value) bool {
	switch that := other.(type) {
	case *wsRefAtVersion:
		return ws.Version() == that.version && ws.Equal(that.ws)
	case *Worksheet:
		return ws == that
	default:
		return false
	}
}

func (value *wsRefAtVersion) diffCompare(other Value) bool {
	switch that := other.(type) {
	case *wsRefAtVersion:
		return value.version == that.version && value.ws.Equal(that.ws)
	case *Worksheet:
		return value.version == that.Version() && value.ws.Equal(that)
	default:
		return false
	}
}

// wsRefAtVersion represents a worksheet reference at a specific version. This
// is only the reference, and not the content of the worksheet at the version.
// Instead, the content is a "head".
type wsRefAtVersion struct {
	ws      *Worksheet
	version int
}

func (_ *wsRefAtVersion) Type() Type {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) Equal(other Value) bool {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) String() string {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) dbWriteValue() string {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) jsonMarshalValue(m *marshaler, b *bytes.Buffer) {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) structScanConvert(ctx *structScanCtx, fieldCtx structScanFieldCtx) (reflect.Value, error) {
	panic("wsRefAtVersion: marker value for diffing only")

}

func (_ *wsRefAtVersion) assignableTo(typ Type) bool {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) selectors() []tSelector {
	panic("wsRefAtVersion: marker value for diffing only")
}

func (_ *wsRefAtVersion) compute(ws *Worksheet) (Value, error) {
	panic("wsRefAtVersion: marker value for diffing only")
}

type change struct {
	before, after Value
}

func (ws *Worksheet) diff() map[int]change {
	allIndexes := make(map[int]bool)
	for index := range ws.orig {
		allIndexes[index] = true
	}
	for index := range ws.data {
		allIndexes[index] = true
	}

	diff := make(map[int]change)
	for index := range allIndexes {
		orig, hasOrig := ws.orig[index]
		data, hasData := ws.data[index]
		if hasOrig && !hasData {
			diff[index] = change{
				before: orig,
				after:  vUndefined,
			}
		} else if !hasOrig && hasData {
			diff[index] = change{
				before: vUndefined,
				after:  data,
			}
		} else if !orig.diffCompare(data) {
			diff[index] = change{
				before: orig,
				after:  data,
			}
		}
	}

	return diff
}

type sliceChange struct {
	deleted []sliceElement
	added   []sliceElement
}

func diffSlices(before, after *Slice) sliceChange {
	var (
		b, a            int
		elementsDeleted []sliceElement
		elementsAdded   []sliceElement
	)
	for b < len(before.elements) && a < len(after.elements) {
		bElement, aElement := before.elements[b], after.elements[a]
		if bElement.rank == aElement.rank {
			if !bElement.value.diffCompare(aElement.value) {
				// we've replaced the value at this rank
				// represent as a delete and an add
				elementsDeleted = append(elementsDeleted, bElement)
				elementsAdded = append(elementsAdded, aElement)
			}
			b++
			a++
		} else if bElement.rank < aElement.rank {
			elementsDeleted = append(elementsDeleted, bElement)
			b++
		} else if aElement.rank < bElement.rank {
			elementsAdded = append(elementsAdded, aElement)
			a++
		}
	}
	for ; b < len(before.elements); b++ {
		elementsDeleted = append(elementsDeleted, before.elements[b])
	}
	for ; a < len(after.elements); a++ {
		elementsAdded = append(elementsAdded, after.elements[a])
	}
	return sliceChange{elementsDeleted, elementsAdded}
}
