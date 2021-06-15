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

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/homelight/worksheets/wstesting"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wstest filename...")
		os.Exit(1)
	}

	var encounteredFailure bool
	for i, filename := range os.Args[1:] {
		if 0 < i {
			fmt.Println()
		}

		if ok := runFeature(filename); !ok {
			encounteredFailure = true
		}
	}

	if encounteredFailure {
		os.Exit(1)
	}
	os.Exit(0)
}

func runFeature(filename string) bool {
	// open doc
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%s\n", filename)
		fmt.Printf("FAIL\t%s\n", err)
		return false
	}
	defer file.Close()

	// read feature
	scenarios, err := wstesting.ReadFeature(bufio.NewReader(file), filename)
	if err != nil {
		fmt.Printf("%s\n", filename)
		fmt.Printf("FAIL\t%s\n", err)
		return false
	}

	// run scenarios
	var (
		currentDir = filepath.Dir(filename)
		ok         = true
	)
	for _, s := range scenarios {
		err := s.Run(wstesting.Context{
			CurrentDir: currentDir,
		})
		if err != nil {
			fmt.Printf("%s\n", s.Name)
			fmt.Printf("FAIL\t%s\n", err)
			ok = false
		}
	}
	return ok
}
