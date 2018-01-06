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
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"gopkg.in/mgutz/dat.v2/sqlx-runner"
)

type DbZuite struct {
	suite.Suite
	db    *runner.DB
	store *DbStore
}

func (s *DbZuite) SetupSuite() {
	// db
	dbUrl := "postgres://ws_user:@localhost/ws_test?sslmode=disable"
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		panic(err)
	}
	s.db = runner.NewDB(db, "postgres")

	// store
	s.store = NewStore(defs)
}

func (s *DbZuite) SetupTest() {
	for table := range tableToEntities {
		_, err := s.db.Exec(fmt.Sprintf("truncate %s", table))
		if err != nil {
			panic(err)
		}
	}
}

func (s *DbZuite) TearDownSuite() {
	err := s.db.DB.Close()
	if err != nil {
		panic(err)
	}
}

func TestRunAllTheDbTests(t *testing.T) {
	suite.Run(t, new(DbZuite))
}
