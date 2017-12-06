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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
)

type localStore struct {
	defs     *Definitions
	filename string
}

// Assert localStore implements the Store interface.
var _ Store = &localStore{}

func newLocalStore(defs *Definitions, filename string) *localStore {
	return &localStore{
		defs:     defs,
		filename: filename,
	}
}

type byName map[string]byId

type byId map[string]map[string]string

func (s *localStore) Load(name, id string) (*Worksheet, error) {
	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		return nil, err
	}

	var byName byName
	err = json.Unmarshal(data, &byName)
	if err != nil {
		return nil, err
	}

	byId, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("worksheet not found %s:%s", name, id)
	}

	wsData, ok := byId[id]
	if !ok {
		return nil, fmt.Errorf("worksheet not found %s:%s", name, id)
	}

	ws, err := s.defs.UnsafeNewUninitializedWorksheet(name)
	if err != nil {
		return nil, err
	}

	for key, value := range wsData {
		index, err := strconv.Atoi(key)
		if err != nil {
			panic(fmt.Sprintf("unexpected %s", err))
		}
		lit, err := NewValue(value)
		if err != nil {
			panic(fmt.Sprintf("unexpected %s", err))
		}
		ws.data[index] = lit
	}

	if err := ws.validate(); err != nil {
		return nil, err
	}

	return ws, nil
}

func (s *localStore) Save(ws *Worksheet) error {
	return fmt.Errorf("not implemented")
}
