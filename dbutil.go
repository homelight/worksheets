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

import runner "github.com/helloeave/dat/sqlx-runner"

func RunTransaction(db *runner.DB, fn func(tx *runner.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.AutoRollback()

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
