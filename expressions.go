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
)

type expression interface {
	Compute(ws *Worksheet) Value
}

// Assert that all expressions implement the expression interface
var _ = []expression{
	&tExternal{},
}

func (e *tExternal) Compute(ws *Worksheet) Value {
	panic(fmt.Sprintf("unresolved plugin in worksheet(%s)", ws.def.name))
}

type ePlugin struct {
	computedBy ComputedBy
}

func (e *ePlugin) Compute(ws *Worksheet) Value {
	args := e.computedBy.Args()
	values := make([]Value, len(args), len(args))
	for i, arg := range args {
		value := ws.MustGet(arg)
		values[i] = value
	}
	return e.computedBy.Compute(values...)
}
