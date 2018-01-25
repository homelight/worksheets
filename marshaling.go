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
	"encoding/json"
	"fmt"
	"strconv"
)

// Assert that Worksheets implement the json.Marshaler interface.
var _ json.Marshaler = &Worksheet{}

func (ws *Worksheet) MarshalJSON() ([]byte, error) {
	m := &marshaler{
		graph: make(map[string][]byte),
	}
	m.marshal(ws)

	var (
		not_first bool
		b         bytes.Buffer
	)
	b.WriteRune('{')
	for id, mashaled := range m.graph {
		if not_first {
			b.WriteRune(',')
		}
		not_first = true

		b.WriteRune('"')
		b.WriteString(id)
		b.WriteString(`":`)
		b.Write(mashaled)
	}
	b.WriteRune('}')
	return b.Bytes(), nil
}

type marshaler struct {
	graph map[string][]byte
}

func (m *marshaler) marshal(ws *Worksheet) {
	if _, ok := m.graph[ws.Id()]; ok {
		return
	}
	m.graph[ws.Id()] = nil

	var (
		not_first bool
		b         bytes.Buffer
	)
	b.WriteRune('{')
	for index, value := range ws.data {
		if not_first {
			b.WriteRune(',')
		}
		not_first = true

		b.WriteRune('"')
		b.WriteString(ws.def.fieldsByIndex[index].name)
		b.WriteString(`":`)
		m.marshalValue(&b, value)
	}
	b.WriteRune('}')
	m.graph[ws.Id()] = b.Bytes()
}

func (m *marshaler) marshalValue(b *bytes.Buffer, value Value) {
	switch v := value.(type) {
	case *Undefined:
		b.WriteString(`"undefined"`)

	case *Text:
		b.WriteString(strconv.Quote(v.value))

	case *Number:
		b.WriteRune('"')
		b.WriteString(value.String())
		b.WriteRune('"')

	case *Bool:
		b.WriteString(strconv.FormatBool(v.value))

	case *slice:
		b.WriteRune('[')
		for i := range v.elements {
			if i != 0 {
				b.WriteRune(',')
			}
			m.marshalValue(b, v.elements[i].value)
		}
		b.WriteRune(']')

	case *Worksheet:
		// 1. We write the ID.
		b.WriteRune('"')
		b.WriteString(v.Id())
		b.WriteRune('"')
		// 2. We ensure this ws is included in the overall marshall.
		m.marshal(v)

	default:
		panic(fmt.Sprintf("unexpected %v of %T", value, value))
	}
}
