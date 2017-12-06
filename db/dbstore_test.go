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

package db

import (
	"github.com/helloeave/worksheets"

	"github.com/stretchr/testify/require"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

func (s *Zuite) TestExample() {
	ws, err := s.store.defs.NewWorksheet("simple")
	require.NoError(s.T(), err)

	err = ws.Set("name", `"Alice"`)
	require.NoError(s.T(), err)

	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		return session.Save(ws)
	})

	var wsFromStore *worksheets.Worksheet
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := s.store.Open(tx)
		var err error
		wsFromStore, err = session.Load("simple", ws.Id())
		return err
	})

	require.Equal(s.T(), `"Alice"`, wsFromStore.MustGet("name").String())
}

func (s *Zuite) MustRunTransaction(fn func(tx *runner.Tx) error) {
	err := RunTransaction(s.db, fn)
	require.NoError(s.T(), err)
}

func RunTransaction(db *runner.DB, fn func(tx *runner.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		defer tx.Rollback()
		return err
	}

	return tx.Commit()
}
