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
	"database/sql"
	"strings"
	"testing"

	"github.com/helloeave/worksheets"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

type Zuite struct {
	suite.Suite
	db    *runner.DB
	store *DbStore
}

const definitions = `
worksheet simple {
	1:name text
	2:age  number(0)
}`

func (s *Zuite) SetupSuite() {
	// db
	dbUrl := "postgres://pascal:@localhost/worksheets_test?sslmode=disable"
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		panic(err)
	}
	s.db = runner.NewDB(db, "postgres")

	// store
	defs, err := worksheets.NewDefinitions(strings.NewReader(definitions))
	if err != nil {
		panic(err)
	}
	s.store = NewStore(defs)
}

func (s *Zuite) TearDownSuite() {
	err := s.db.DB.Close()
	if err != nil {
		panic(err)
	}
}

func TestRunAllTheTests(t *testing.T) {
	suite.Run(t, new(Zuite))
}
