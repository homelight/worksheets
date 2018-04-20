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
	"math"
	"strings"

	"github.com/helloeave/dat/sqlx-runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestComputedBy_externalComputedBy() {
	cases := []struct {
		def         string
		opt         *Options
		expectedErr string
	}{
		{
			`type simple worksheet {
				1:hello_name text computed_by { external }
			}`,
			nil,
			"simple.hello_name: missing plugin for external computed_by",
		},
		{
			`type simple worksheet {}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"not_so_simple": map[string]ComputedBy{
						"unknown_name": nil,
					},
				},
			},
			"plugins: unknown worksheet(not_so_simple)",
		},
		{
			`type simple worksheet {}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"unknown_name": nil,
					},
				},
			},
			"plugins: unknown field simple.unknown_name",
		},
		{
			`type simple worksheet {
				1:name text
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": nil,
					},
				},
			},
			"plugins: field simple.name not externally defined",
		},
		{
			`type simple worksheet {
				1:name text computed_by { external }
				2:age number[0]
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": sayAlice([]string{}),
					},
				},
			},
			"simple.name has no dependencies",
		},
		{
			`type simple worksheet {
				1:name text computed_by { external }
				2:age number[0]
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"simple": map[string]ComputedBy{
						"name": sayAlice([]string{"agee"}),
					},
				},
			},
			"simple.name references unknown arg agee",
		},
		{
			`type parent worksheet {
				1:child child
				2:name text computed_by { external }
			}
			type child worksheet {
				3:field text
			}`,
			&Options{
				Plugins: map[string]map[string]ComputedBy{
					"parent": map[string]ComputedBy{
						"name": sayAlice([]string{"child.not_field"}),
					},
				},
			},
			"parent.name references unknown arg child.not_field",
		},
	}
	for _, ex := range cases {
		var opts []Options
		if ex.opt != nil {
			opts = append(opts, *ex.opt)
		}
		_, err := NewDefinitions(strings.NewReader(ex.def), opts...)
		if assert.Error(s.T(), err) {
			require.Equal(s.T(), ex.expectedErr, err.Error())
		}
	}

}

func (s *Zuite) TestComputedBy_externalComputedByPlugin() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{"age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	err = ws.Set("name", NewText("Alex"))
	if assert.Error(s.T(), err) {
		require.Equal(s.T(), "cannot assign to computed field name", err.Error())
	}
}

type sayAlice []string

var _ ComputedBy = sayAlice([]string{})

func (sa sayAlice) Args() []string {
	return sa
}

func (sa sayAlice) Compute(values ...Value) Value {
	if _, ok := values[0].(*Undefined); ok {
		return vUndefined
	}
	return NewText("Alice")
}

type fullName []string

var _ ComputedBy = fullName([]string{})

func (fn fullName) Args() []string {
	return fn
}

func (fn fullName) Compute(values ...Value) Value {
	var firstName, lastName string
	switch t := values[0].(type) {
	case *Text:
		firstName = t.value
	case *Undefined:
		return vUndefined
	}
	switch t := values[1].(type) {
	case *Text:
		lastName = t.value
	case *Undefined:
		return vUndefined
	}
	return NewText(fmt.Sprintf("%s %s", firstName, lastName))
}

type age []string

var _ ComputedBy = age([]string{})

func (fn age) Args() []string {
	return fn
}

func (fn age) Compute(values ...Value) Value {
	if _, ok := values[0].(*Undefined); ok {
		return vUndefined
	}
	birthYear := values[0].(*Number).value
	return NewNumberFromInt64(2018 - birthYear)
}

type bio []string

var _ ComputedBy = bio([]string{})

func (fn bio) Args() []string {
	return fn
}

func (fn bio) Compute(values ...Value) Value {
	var fullName string
	var birthYear, age int64
	switch t := values[0].(type) {
	case *Text:
		fullName = t.value
	case *Undefined:
		fullName = ""
	}
	switch t := values[1].(type) {
	case *Number:
		birthYear = t.value
	case *Undefined:
		birthYear = 0
	}
	switch t := values[2].(type) {
	case *Number:
		age = t.value
	case *Undefined:
		age = 0
	}

	return NewText(fmt.Sprintf("%s, age %d, born in %d", fullName, age, birthYear))
}

func (s *Zuite) TestComputedBy_externalGood() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"simple": map[string]ComputedBy{
				"name": sayAlice([]string{"age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {
		1:name text computed_by { external }
		2:age number[0]
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	require.False(s.T(), ws.MustIsSet("name"))

	ws.MustSet("age", MustNewValue("73"))
	require.Equal(s.T(), `"Alice"`, ws.MustGet("name").String())
}

func (s *Zuite) TestComputedBy_externalGoodComplicated() {
	opt := Options{
		Plugins: map[string]map[string]ComputedBy{
			"complicated": map[string]ComputedBy{
				"full_name": fullName([]string{"first_name", "last_name"}),
				"age":       age([]string{"birth_year"}),
				"bio":       bio([]string{"full_name", "birth_year", "age"}),
			},
		},
	}
	defs, err := NewDefinitions(strings.NewReader(`type complicated worksheet {
		1:first_name text
		2:last_name text
		3:full_name text computed_by { external }
		4:birth_year number[0]
		5:age number[0] computed_by { external }
		6:bio text computed_by { external }
	}`), opt)
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("complicated")

	require.False(s.T(), ws.MustIsSet("full_name"))

	ws.MustSet("first_name", NewText("Alice"))
	ws.MustSet("last_name", NewText("Maters"))
	ws.MustSet("birth_year", MustNewValue("1945"))
	require.Equal(s.T(), `"Alice Maters"`, ws.MustGet("full_name").String())
	require.Equal(s.T(), `73`, ws.MustGet("age").String())
	require.Equal(s.T(), `"Alice Maters, age 73, born in 1945"`, ws.MustGet("bio").String())
}

func (s *Zuite) TestComputedBy_simpleExpressionsInWorksheet() {
	defs, err := NewDefinitions(strings.NewReader(`type simple worksheet {
		1:age number[0]
		2:age_plus_two number[0] computed_by { return age + 2 }
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	ws.MustSet("age", MustNewValue("73"))
	require.Equal(s.T(), "75", ws.MustGet("age_plus_two").String())
}

func (s *Zuite) TestComputedBy_cyclicEditsIfNoIdentCheck() {
	defs, err := NewDefinitions(strings.NewReader(`type cyclic_edits worksheet {
		1:right bool
		2:a bool computed_by {
			return b || right
		}
		3:b bool computed_by {
			return a || !right
		}
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("cyclic_edits")

	ws.MustSet("right", MustNewValue("true"))
	require.Equal(s.T(), "true", ws.MustGet("right").String(), "right")
	require.Equal(s.T(), "undefined", ws.MustGet("a").String(), "a")
	require.Equal(s.T(), "undefined", ws.MustGet("b").String(), "b")
}

var defsCrossWs = `
type parent worksheet {
	1:child_amount number[2] computed_by {
		return child.amount
	}
	2:child child
}

type child worksheet {
	5:amount number[2]
}`

func (s *Zuite) TestComputedBy_simpleCrossWsParentPointers() {
	parent := s.defsCrossWs.MustNewWorksheet("parent")
	child := s.defsCrossWs.MustNewWorksheet("child")
	forciblySetId(parent, "parent-id")

	require.Len(s.T(), child.parents, 0)
	require.Len(s.T(), child.parents["parent"], 0)
	require.Len(s.T(), child.parents["parent"][2], 0)

	parent.MustSet("child", child)
	require.Len(s.T(), child.parents, 1)
	require.Len(s.T(), child.parents["parent"], 1)
	require.Len(s.T(), child.parents["parent"][2], 1)
	require.True(s.T(), child.parents["parent"][2]["parent-id"] == parent)

	parent.MustUnset("child")
	require.Len(s.T(), child.parents, 0)
	require.Len(s.T(), child.parents["parent"], 0)
	require.Len(s.T(), child.parents["parent"][2], 0)
}

func (s *Zuite) TestComputedBy_simpleCrossWsExample() {
	parent := s.defsCrossWs.MustNewWorksheet("parent")

	child := s.defsCrossWs.MustNewWorksheet("child")
	child.MustSet("amount", MustNewValue("1.11"))
	parent.MustSet("child", child)
	require.Equal(s.T(), "1.11", parent.MustGet("child_amount").String())

	child.MustSet("amount", MustNewValue("2.22"))
	require.Equal(s.T(), "2.22", parent.MustGet("child_amount").String())

	parent.MustUnset("child")
	require.Equal(s.T(), "undefined", parent.MustGet("child_amount").String())
}

type sumPlugin string

// Assert that sumPlugin implements the ComputedBy interface.
var _ ComputedBy = sumPlugin("")

func (p sumPlugin) Args() []string {
	return []string{string(p)}
}

func (p sumPlugin) Compute(values ...Value) Value {
	slice := values[0].(*Slice)
	sum := MustNewValue("0").(*Number)
	for _, elem := range slice.Elements() {
		if num, ok := elem.(*Number); ok {
			sum = sum.Plus(num)
		} else {
			return vUndefined
		}
	}
	return sum
}

var defsCrossWsThroughSlice = `
type parent worksheet {
	10:sum_child_amount number[2] computed_by {
		external
	}
	20:children []child
}

type child worksheet {
	50:amount number[2]
}`

var defsCrossWsThroughSliceOptions = Options{
	Plugins: map[string]map[string]ComputedBy{
		"parent": {
			"sum_child_amount": sumPlugin("children.amount"),
		},
	},
}

func (s *Zuite) TestComputedBy_crossWsThroughSliceParentPointers() {
	parent := s.defsCrossWsThroughSlice.MustNewWorksheet("parent")
	child1 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
	child2 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
	forciblySetId(parent, "parent-id")

	require.Len(s.T(), child1.parents, 0)
	require.Len(s.T(), child2.parents, 0)

	parent.MustAppend("children", child1)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][20], 1)
	require.True(s.T(), child1.parents["parent"][20]["parent-id"] == parent)
	require.Len(s.T(), child2.parents, 0)

	parent.MustAppend("children", child2)
	require.Len(s.T(), child1.parents, 1)
	require.Len(s.T(), child1.parents["parent"], 1)
	require.Len(s.T(), child1.parents["parent"][20], 1)
	require.True(s.T(), child1.parents["parent"][20]["parent-id"] == parent)
	require.Len(s.T(), child2.parents, 1)
	require.Len(s.T(), child2.parents["parent"], 1)
	require.Len(s.T(), child2.parents["parent"][20], 1)
	require.True(s.T(), child2.parents["parent"][20]["parent-id"] == parent)

	parent.Del("children", 0)
	require.Len(s.T(), child1.parents, 0)
	require.Len(s.T(), child2.parents, 1)
	require.Len(s.T(), child2.parents["parent"], 1)
	require.Len(s.T(), child2.parents["parent"][20], 1)
	require.True(s.T(), child2.parents["parent"][20]["parent-id"] == parent)

	parent.Del("children", 0)
	require.Len(s.T(), child1.parents, 0)
	require.Len(s.T(), child2.parents, 0)
}

func (s *Zuite) TestComputedBy_crossWsThroughSliceExample() {
	parent := s.defsCrossWsThroughSlice.MustNewWorksheet("parent")

	require.Equal(s.T(), "0", parent.MustGet("sum_child_amount").String())

	child1 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
	child1.MustSet("amount", MustNewValue("1.11"))
	parent.MustAppend("children", child1)
	require.Equal(s.T(), "1.11", parent.MustGet("sum_child_amount").String())

	child2 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
	child2.MustSet("amount", MustNewValue("2.22"))
	parent.MustAppend("children", child2)
	require.Equal(s.T(), "3.33", parent.MustGet("sum_child_amount").String())

	parent.Del("children", 0)
	require.Equal(s.T(), "2.22", parent.MustGet("sum_child_amount").String())

	parent.Del("children", 0)
	require.Equal(s.T(), "0", parent.MustGet("sum_child_amount").String())
}

func (s *Zuite) TestComputedBy_crossWs_parentsRefsPersistence() {
	store := NewStore(s.defsCrossWs)

	// We create a parent, pointing to a child.
	var (
		parentId = "aaaaaaaa-9be5-41e4-9b56-787f52f5a198"
		childId  = "bbbbbbbb-9be5-41e4-9b56-787f52f5a198"
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		parent := s.defsCrossWs.MustNewWorksheet("parent")
		forciblySetId(parent, parentId)
		child := s.defsCrossWs.MustNewWorksheet("child")
		forciblySetId(child, childId)
		child.MustSet("amount", MustNewValue("6.66"))
		parent.MustSet("child", child)
		session := store.Open(tx)
		_, err := session.Save(parent)
		return err
	})

	// 1. Ensure parent pointers are properly stored on save.
	snap := s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          childId,
			ParentId:         parentId,
			ParentFieldIndex: 2,
		},
	}, snap.parentsRecs)

	// 2. Ensure parent pointers (and parent worksheets) are correctly loaded.
	var childParentsAfterLoad parentsRefs
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		child, err := session.Load(childId)
		if err != nil {
			return err
		}
		childParentsAfterLoad = child.parents
		return nil
	})

	require.Len(s.T(), childParentsAfterLoad, 1)
	require.Len(s.T(), childParentsAfterLoad["parent"], 1)
	require.Len(s.T(), childParentsAfterLoad["parent"][2], 1)
	require.NotNil(s.T(), childParentsAfterLoad["parent"][2][parentId])
	require.Equal(s.T(), `6.66`, childParentsAfterLoad["parent"][2][parentId].MustGet("child_amount").String())

	// 3. Ensure that when a ref is removed from the parent, the parent record
	// is properly removed (even when the child is not loaded).
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustUnset("child")
		_, err = session.Update(parent)
		return err
	})

	snap = s.snapshotDbState()

	require.Empty(s.T(), snap.parentsRecs)
}

func (s *Zuite) TestComputedBy_crossWs_twoParentsOneChildRefsPersistence() {
	store := NewStore(s.defsCrossWs)

	// We create two parents, pointing to the same child.
	var (
		parent1Id = "aaaaaaaa-9be5-41e4-9b56-787f52f5a198"
		parent2Id = "bbbbbbbb-9be5-41e4-9b56-787f52f5a198"
		childId   = "cccccccc-9be5-41e4-9b56-787f52f5a198"
	)
	s.MustRunTransaction(func(tx *runner.Tx) error {
		parent1 := s.defsCrossWs.MustNewWorksheet("parent")
		forciblySetId(parent1, parent1Id)
		parent2 := s.defsCrossWs.MustNewWorksheet("parent")
		forciblySetId(parent2, parent2Id)
		child := s.defsCrossWs.MustNewWorksheet("child")
		forciblySetId(child, childId)
		child.MustSet("amount", MustNewValue("6.66"))
		parent1.MustSet("child", child)
		parent2.MustSet("child", child)

		// Since parent1 -> child -> parent2, when saving parent1, we also
		// save parent2!
		_, err := store.Open(tx).Save(parent1)
		return err
	})

	// 1. Ensure parent pointers are properly stored on save.
	snap := s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          childId,
			ParentId:         parent1Id,
			ParentFieldIndex: 2,
		},
		{
			ChildId:          childId,
			ParentId:         parent2Id,
			ParentFieldIndex: 2,
		},
	}, snap.parentsRecs)

	// 2. Ensure that when a ref is removed from a parent, the parent record
	// is properly removed (even when the child is not loaded), and that no
	// other parent record is touched.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent1, err := session.Load(parent1Id)
		if err != nil {
			return err
		}
		parent1.MustUnset("child")
		_, err = session.Update(parent1)
		return err
	})

	snap = s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          childId,
			ParentId:         parent2Id,
			ParentFieldIndex: 2,
		},
	}, snap.parentsRecs)
}

func (s *Zuite) TestComputedBy_crossWs_parentWithSlicesRefsPersistence1() {
	var (
		store    = NewStore(s.defsCrossWsThroughSlice)
		parentId = "aaaaaaaa-9be5-41e4-9b56-787f52f5a198"
		child1Id = "bbbbbbbb-9be5-41e4-9b56-787f52f5a198"
		child2Id = "cccccccc-9be5-41e4-9b56-787f52f5a198"
	)

	// We create a parent, pointing to a child through a slice.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		parent := s.defsCrossWsThroughSlice.MustNewWorksheet("parent")
		forciblySetId(parent, parentId)
		child1 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
		forciblySetId(child1, child1Id)
		child1.MustSet("amount", MustNewValue("6.66"))
		parent.MustAppend("children", child1)
		session := store.Open(tx)
		_, err := session.Save(parent)
		return err
	})

	// 1. Ensure parent pointers are properly stored on save.
	snap := s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          child1Id,
			ParentId:         parentId,
			ParentFieldIndex: 20,
		},
	}, snap.parentsRecs)

	// 2. Add another child, ensure the new ref is also recorded.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		child2 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
		forciblySetId(child2, child2Id)
		child2.MustSet("amount", MustNewValue("7.77"))
		parent.MustAppend("children", child2)
		_, err = session.Update(parent)
		return err
	})

	snap = s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          child1Id,
			ParentId:         parentId,
			ParentFieldIndex: 20,
		},
		{
			ChildId:          child2Id,
			ParentId:         parentId,
			ParentFieldIndex: 20,
		},
	}, snap.parentsRecs)

	// 3. Remove a child, ensure ref is removed as well.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		parent.MustDel("children", 0)
		_, err = session.Update(parent)
		return err
	})

	snap = s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          child2Id,
			ParentId:         parentId,
			ParentFieldIndex: 20,
		},
	}, snap.parentsRecs)
}

func (s *Zuite) TestComputedBy_crossWs_parentWithSlicesRefsPersistence2() {
	var (
		store    = NewStore(s.defsCrossWsThroughSlice)
		parentId = "aaaaaaaa-9be5-41e4-9b56-787f52f5a198"
		childId  = "bbbbbbbb-9be5-41e4-9b56-787f52f5a198"
	)

	// We create a parent ws, and a child ws, but we do not connect them yet.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		parent := s.defsCrossWsThroughSlice.MustNewWorksheet("parent")
		forciblySetId(parent, parentId)

		child := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
		forciblySetId(child, childId)
		child.MustSet("amount", MustNewValue("6.66"))

		session := store.Open(tx)
		if _, err := session.Save(parent); err != nil {
			return err
		}
		if _, err := session.Save(child); err != nil {
			return err
		}
		return nil
	})

	// In a subsequent transaction, we connect the parent to the child.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		child, err := session.Load(childId)
		if err != nil {
			return err
		}

		parent.MustAppend("children", child)
		_, err = session.Update(parent)
		return err
	})

	// Now, ensure parent pointers are properly stored on save.
	snap := s.snapshotDbState()

	require.Equal(s.T(), []rParent{
		{
			ChildId:          childId,
			ParentId:         parentId,
			ParentFieldIndex: 20,
		},
	}, snap.parentsRecs)
}

func (s *Zuite) TestComputedBy_crossWs_updateOfChildCarriesToParent() {
	var (
		store    = NewStore(s.defsCrossWsThroughSlice)
		parentId = "aaaaaaaa-9be5-41e4-9b56-787f52f5a198"
		child1Id = "bbbbbbbb-9be5-41e4-9b56-787f52f5a198"
		child2Id = "cccccccc-9be5-41e4-9b56-787f52f5a198"
	)

	// We create a parent, pointing to two children through a slice.
	var childrenSliceId string
	s.MustRunTransaction(func(tx *runner.Tx) error {
		parent := s.defsCrossWsThroughSlice.MustNewWorksheet("parent")
		child1 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
		child2 := s.defsCrossWsThroughSlice.MustNewWorksheet("child")
		forciblySetId(parent, parentId)
		forciblySetId(child1, child1Id)
		forciblySetId(child2, child2Id)
		child1.MustSet("amount", MustNewValue("6.66"))
		child2.MustSet("amount", MustNewValue("7.77"))
		parent.MustAppend("children", child1)
		parent.MustAppend("children", child2)
		childrenSliceId = parent.data[20].(*Slice).id
		session := store.Open(tx)
		_, err := session.Save(parent)
		return err
	})

	// Load only child2, update its amount, persist. Then, in a separate
	// transaction, load parent, and observe its sum being properly updated.
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		child2, err := session.Load(child2Id)
		if err != nil {
			return err
		}
		child2.MustSet("amount", MustNewValue("8.88"))
		_, err = session.Update(child2)
		return err
	})

	var sumOfChildren Value
	s.MustRunTransaction(func(tx *runner.Tx) error {
		session := store.Open(tx)
		parent, err := session.Load(parentId)
		if err != nil {
			return err
		}
		sumOfChildren = parent.MustGet("sum_child_amount")
		return nil
	})
	require.Equal(s.T(), `15.54` /* 6.66 + 8.88 */, sumOfChildren.String())

	snap := s.snapshotDbState()

	require.Equal(s.T(), []rValueForTesting{
		// parent's values
		{
			WorksheetId: parentId,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       parentId,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: parentId,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: parentId,
			Index:       10,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `14.43`, // 6.66 + 7.77
		},
		{
			WorksheetId: parentId,
			Index:       10,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `15.54`, // 6.66 + 8.88
		},
		{
			WorksheetId: parentId,
			Index:       20,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `[:2:` + childrenSliceId,
		},

		// child1's values
		{
			WorksheetId: child1Id,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       child1Id,
		},
		{
			WorksheetId: child1Id,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `1`,
		},
		{
			WorksheetId: child1Id,
			Index:       50,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       `6.66`,
		},

		// child2's values
		{
			WorksheetId: child2Id,
			Index:       indexId,
			FromVersion: 1,
			ToVersion:   math.MaxInt32,
			Value:       child2Id,
		},
		{
			WorksheetId: child2Id,
			Index:       indexVersion,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `1`,
		},
		{
			WorksheetId: child2Id,
			Index:       indexVersion,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `2`,
		},
		{
			WorksheetId: child2Id,
			Index:       50,
			FromVersion: 1,
			ToVersion:   1,
			Value:       `7.77`,
		},
		{
			WorksheetId: child2Id,
			Index:       50,
			FromVersion: 2,
			ToVersion:   math.MaxInt32,
			Value:       `8.88`,
		},
	}, snap.valuesRecs)
}

func (s *Zuite) TestComputedBy_computedByOnNewInstance() {
	defs := MustNewDefinitions(strings.NewReader(`
	type sum_should_be_zero_on_new worksheet {
		1:nums []number[0]
		2:sum number[0] computed_by {
			return sum(nums)
		}
	}`))

	ws := defs.MustNewWorksheet("sum_should_be_zero_on_new")
	require.Equal(s.T(), NewNumberFromInt(0), ws.MustGet("sum"))
}
