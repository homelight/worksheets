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

// TODO(pascal): these interfaces will need to be exported...

// rType represents a runtime type.
type rType interface {
	// AssignableTo reports whether a value of the type is assignable to type u.
	AssignableTo(u rType) bool

	// String returns a string representation of the type.
	String() string
}

// rValue represents a runtime value.
type rValue interface {
	// Type returns this value's type.
	Type() rType

	// String returns a string representation of the value.
	String() string
}
