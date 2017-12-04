# Worksheets -- Overview

[![CircleCI](https://circleci.com/gh/helloeave/worksheets.svg?style=svg&circle-token=273512d656713e7f4a7ed0d464aa297999c54f0b)](https://circleci.com/gh/helloeave/worksheets)
[![GoDoc](https://godoc.org/github.com/helloeave/worksheets?status.svg)](https://godoc.org/github.com/helloeave/worksheets)

Let's start with a motivating example, easily representing a borrower's legal name, date of birth, and determining if they are of legal age to get a mortgage.

We start by giving the definition for what data describe a `borrower` worksheet

	worksheet borrower {
		1:first_name text
		2:last_name text
		3:dob date
		4:can_take_mortgage computed {
			return dob > now + 18 years
		}
	}

From this definition, we generate Golang hooks to manipulte borrowers' worksheets, store and retrieve them

	joey := worksheet.Create("borrower")
	joey.SetText("first_name", "Joey")
	joey.SetText("last_name", "Pizzapie")
	joey.SetDate("dob", 1980, 5, 23)

We can query worksheet

	joey.GetBool("can_take_mortgage")

Store the worksheet

	bytes, err := joey.Marshal()

And retrieve the worksheet

	joey, err := worksheet.Unmarshal("borrower", bytes)

# Worksheet Definition

All centers around the concept of a `worksheet` which is constituted of typed named fields

	worksheet person {
		1:age number(0)
		2:first_name text
	}

The general syntax for a field is

	index:name type [extras]

(We explain the need for the index in the storage section. Those familiar with Thrift or Protocol Buffers can see the parralel with these data representation tools.)

## Input Fields

The simplest fields we have are there to store values. In the example above, both `age` and `first_name` are input fields. These can be edited and read freely.

## Constrained Fields

We can also constrain fields

	3:social_security_number number(0) constrained_by {
		100_00_0000 <= social_security_number
		social_security_number <= 999_99_9999
	}

When fields are constrained, edits which do not satisfy the constraint are rejected.

## Computed Fields

Instead of being input fields, we can have output fields, or computed fields

	1:date_of_birth date
	2:age number(0) computed_by {
		return (now - date) in years
	}

(We discuss the syntax in which expressions can be written in a later section.)

Computed fields are determined when their inputs changes, and then materialized. Said another way, if any of the input of a computed field changes, its value is re-computed, and then the resulting value is stored into the worksheet. Computed fields are not computed on the fly, they are only computed in an edit cycle.

## Identity

All worksheets have a unique identifier

	identity text

Which is set upon creation, and cannot be ever edited.

## Versioning

All worksheets are versionned, with their version number starting at `1`. The version is stored in a field present on all worksheets

	version numeric(0)

This `version` field is set to `1` upon creation, and incremented on every edit. Versions are used to detect concurrent edits, and abort an edit that was done on an older version of the worksheet than the one it would now be applied to. Edits are discussed in greater detail later.

## Data Representation

### Base Types

The various base types are

| Type        | Represents |
|-------------|------------|
| `bool`      | Booleans. |
| `text`      | Text of arbitrary length. |
| `number(n)` | Numbers with precision of _n_ decimal places. |
| `time`      | Instant in time. (Time zone independent.) |
| `date`      | Specific date like 7/20/1969. (Time zone dependent.) |

All base types represent optional values, i.e. `bool` covers three values `true`, `false`, and `undefined`.

As described later, all operations over base types have an interpretation with respect to `undefined`. It is often simply treated as an absorbing element such that `v OP undefined = undefined OP v = undefined`.

### Enums

While the language does not have a specific support for enums, these can be described easily with the use of single field worksheets

	worksheet name_suffixes {
		1:suffix text constrained_by {
			return suffix in [
				"Jr.",
				"Sr."
				...
			]
		}
	}

Which can then be used

	worksheet borrower {
		...

		3:first_name text
		4:last_name text
		5:suffix name_suffixes

		...
	}

And the Golang hooks allow handling of single field worksheets in a natural way, by automatically 'boxing'

	borrower.SetText("suffix", "Jr.")

Or 'unboxing'

	borrower.GetText("suffix")

### Numbers

Numbers in worksheets have a fixed point precision (e.g. 2 decimal places), and all operations on numbers guarantee the precision to be strictly preserved.

Since we intend to eventally have a statically typed langugage, we choose statically determined rules for data flow, though we expect initial implementations to be dynamic.

#### Syntax

Number literals are represented in base 10, and may contain any number of underscores (`_`) to be added for clarity. We can write `123098` or `123_098`, and we could even write `1____2_3______098`.

The decimal portion of number literal is automatically expanded to fit the type it is being assigned to. For instance, if we store `5.2` in a field `number(5)`, this would yield `5.200_00`.

#### Handling Precision, and Rounding

Precision expansion is allowed, such that you can store a `number(n)` into a `number(m)` if `n` is smaller than `m`. In such cases, we simply do a precision expansion.

Loss of precision however needs to be explicitely handled.

For instance, with the field `age number(0)`, the expression

	age, _ = 5.200 round down

would yield `5` in the `age` field.

Rounding modes supported are `up`, `down`, `even`.

#### Addition, Substraction

When adding or subtracting numbers, the rule for decimal treatment is

`number(n) op number(m)` yields `number(max(n, m))`

where `op` is either `+` or `-`.

So for instance `5.03 + 6.000` would yield `11.030` as a `number(3)` even though it could be dynamically represented as a `number(2)`.

#### Multiplication

When multiplying, the rule for decimal treatment is

`number(n) * number(m)` yields `number(n + m)`

So for instance `5.30 * 6.0` would yield `31.800` as a `number(3)` even though it could be dynamically represented as a `number(1)`.

#### Explicit Rounding When Dividing

When dividing, a rounding mode must always be provided such that the syntax for division is `v1 / v2 round mode`.

For instance, consider the following example. We need to create a `map[repayment]` to represent repayment of $700 in yearly taxes, over a 12 months period. If we were to split in equal parts, we would need to pay $58.33... which is not feasibly. Instead, here we force ourselves to round to cents, i.e. a `number(2)`.

	first_month = month of closing_date + 1
	total_paid := 0
	for current_month := first_month; current_month < first_month + 1 year; current_month++ {
		mp := yearly_taxes / 12 round down
		total_paid += mp
		payment_schedule << payment {
			month = current_month
			amount = mp
		}
	}

	remainder := yearly_taxes - total_paid
	switch remainder % 2 {
		case 0:
			payment_schedule[first_month + 6 months].amount += remainder / 2
			payment_schedule[first_month + 12 months].amount += remainder / 2
		case 1:
			payment_schedule[first_month + 12 months].amount += remainder
	}

### Time and Date

* time as instant in time, timezone less concept, only display (because it can be lossy) needs timezone in some cases

* date as a timezone depenedent concept, range of time, so 9/1/2017 ET delineates a specific range

* date without timezone, to represent input of borrowers, needs to then be interpreted in the context of a timezone to be useful

also, how do we convert to date objects

date(yyyy, mm, dd, tz) could be a function taking three numbers (year, month, day) and a timezone and produce a date object

some_time.date(tz) -> yields date in timezone

some_date.year / .month / .day -> yields sub-component (since date is already tz dependent, this is good)

## Keyed Worksheets, Maps, and Tuples

In addition to the structures covered earlier, we have

	map[W]

which represents a collection of worksheets `W`, indexed by the key of `W`.

### Tuples

As we shall see in the next section, worksheets' keys can be one or multiple values joined together in N-tuples. To represent this, we introduce

	tuple[T1, ..., Tn]

NOTE #1: We don't want to allow `tuple[map[foo]]`, so how do we differentiate the simple types, from more complex types like maps? Maps can be the type of fields, but the type which can go in tuples is a subset of that. Maybe 'base type' as `S` and field type `T` is a sufficient distinction? Then maps and tuples would be field types, one over worksheets, the other over only base types.

NOTE #2: It would make sense to have undefindeness defined only over base types, hence a tupe would never itself be `undefined`, rather all of its components can be.

### Keyed Worksheets

Worksheets present in maps must keyed: one or more fields of the worksheet serve as a unique key for the purpose of the mapping

    keyed_by {
		first_name
		last_name
    }

or

	keyed_by identity

When a worksheet is added in a map for the first time, all the fields covered by the key are frozen and are not allowed to be mutated later. This means that if a computed field is part of a key, all the inputs to this computed fields will be frozen.

NOTE: To be consistent with above, we could only allow fields of base types to be part of a key.

### operations on maps

- add: can only add if key not present (no implicit replace)
- delete: should we be strict about deletion, i.e. require the key to be present, or not?
- check presence, like `_, ok := the_map[the_key]`
- size: gets the size of the map (note: should we keep len to be closer to go?)
- iterations `for k := range range the_map {` or `for k, v := range the_map`

when iterating over maps, iteration order is the order in which items were added in (essentially a hash map + linked list of the elements)

# Editing Worksheets

There are a three basic steps to editing a worksheet

1. Proposing an edit (only inputs)
2. Determining the actual edit (fixed-point calculation)
3. Applying the actual edit (atomic)

Let's go through them with the help of a contrived example

	worksheet borrower {
		1:name text
		2:greeting text computed {
			return "Hello, " + name
		}
	}

We can _propose_ the edit

	set name "Joey"

Which yields the _actual_ edit

	[ on borrower(the-id-here) @ version 5 ]
	[ set version 6                        ]
	set name "Joey"
	set greeting "Hello, Joey"

And when _applied_ mutates the worksheet as intended. (In future examples, we omit the concurrent modification part of edits.)

## Edit Blocks, and Individual Edits

We call the set of edits an _edit block_, which is itself constituted of _individual edits_.

Some of the possible individual edits are

- Setting a field to a specific value
- Unsettting a field, i.e. settting it to `undefined`
- Adding, or removing to a map

In a given edit block, fields can be edited only once, and we allow only one operation per map key. (Adding a worksheet into a map with the contains another worksheet with the same key causes a replace.) As such, the order in which edits are applied is semantically irrelevant.

## Proposed Edits, Tentative Edits, and Actual Edits

Proposed edit blocks can modify any number of inputs in a worksheet. However, as described earlier, computed fields cannot be modified directly.

When a proposed edit block is applied to a worksheet, all computed fields whose inputs were modified are re-computed. This yields a new tentative edit block. In the case where computed fields depend on other computed fields, the process may need to be repeated until we reach a 'fixed point' to get the actual edit block.

Let's consider the worksheet

	worksheet borrower {
		1:name text
		2:name_short text computed {
			if len(name) > 5 {
				return substr(name, 5)
			}
			return name
		}
		3:greeting text computed {
			return "Hello, " + name_short
		}
	}

And the proposed edit

	set name "Samantha"

The first tentative edit would be

	set name "Samantha"
	set name_short "Saman"

The second (and final) edit would be

	set name "Samantha"
	set name_short "Saman"
	set greeting "Hello, Saman"

## Applying Edits

When edit blocks are applied to a worksheet, all individual edits are applied atomically, i.e. _all_ individual edits succeed or _none_ of the individual edits succeed.

## Edit Reacting Code

We also make it possible for user specified code to intercept the fixed-point calculation introduce any edit. This can be useful for cases where you are unable to express computed fields, or cases where doing so in the worksheet language would be unclear. One example could be setting various dates when certain events occur.

Example

	func (myEditor *) OnEdit(current Worksheet, proposed_edit Edit) (Edit, error) {
		if current.GetText("name") == "Joey" {
			return proposed_edit.SetText("name", "Joey Pizzapie"), nil
		}
		return proposed_edit, nil
	}

NOTE: We'd want the `Edit` struct to have sufficient introspection such that we can clearly write cases like the TRID Rule where we need to capture the _first_ time all 6 fields are set on a specific worksheet. Maybe we should also provide a pre/post worksheet with the state before any edit, the state after if the edit were to succeed as is? Need to think through what that code would look like and design the hook with that in mind.

One idea would be to be able to verify 'is any of these six fields being modified?', and 'is the resulting edit one where all six fields are complete?', and 'has the trid rule triggered date been set?'.

	if current.GetDate("trid_rule_triggered").IsUndefined() {
		if proposed_edit.IsSetting("ssn")
		proposed_edit.IsSetting("...") ||
		proposed_edit.IsSetting("...") ||
		... {
			if !current.GetText("ssn").IsUndefined() &&
			... {

			}
		}
	}

Though we'd not even need to verify whether these fields are part of the edit, it's implicit, and more 'underlying structure proof' not too. By 'nuderlying structure proof' we mean that the code assumes less about the internal structure of the fields, they could be inputs or computed fields, and we wouldn't really care.

## Preventing Unstable Edits

Assume we have the worksheet

	worksheet cyclic_edits {
		1:right bool
		2:wrong computed {
			return !right
		}
	}

And we propose the edit

	set right false

Due to the computed field, this would yield 'actual edit #1'

	set right false
	set wrong true

Now, assume that we have edit reacting code which flips these fields around

	if wrong {
		set right true
	} else {
		set right false
	}

We would then yield the 'actual edit #2'

	set right true
	set wrong true

And, due to the computed field, this would yield 'actual edit #3'

	set right true
	set wrong false

Further modified via the event reacting code into 'actual edit #4'

	set right false
	set wrong false

And again, due to the computed field, this would yield 'actual edit #5'

	set right false
	set wrong true

Which is in fact the same as 'actual edit #1'. We have created an infinite loop in the fixed-point calculation!

To prevent such 'unstable edits', we detect cycles of edits, and error out, hence rejecting the proposed edit as being invalid. (Implementation note: this must be done by comparing worksheets', not edits, since two different edits can yield the same worksheet transformation.)

# Storing Worksheets

- storage of worksheets can be totally orthogonal from the system itself
- simply need to be able to walk worksheets, and reconstruct them
- basically marshalling and unmarshalling primitives, rather than concrete implementations
- nice direction, would still want to provide base binary representation out of the box

- of the worksheets
- of edits (i.e. the redo log)
- marshalling: models can be saved in same format as thrift (hence, need index marker for each values)

- discuss uniqueness, GUID uniqueness out of the box, if need global uniqueness, needs to be provided

# Computational Model of Computed Fields, and Constrained Fields

- all values are optional

- operations take into account optionality
  e.g. undefined + 6 yields undefined

- no cycles in function graphs, no recursion, etc. but must be without cycles (outputs from inputs)

# Privacy, and Data Protection

_to cover_

## ACLs
## Encyption
## Audit Trail

# Introspection, and Reflection

- query an input and get concrete AST of how it is calculated from all raw values
- query an input to see every value it flows into, i.e. all computed fields using this input

# Implementation Notes

## Efficient Edits

ideas from

- https://en.wikipedia.org/wiki/Rete_algorithm
- https://en.wikipedia.org/wiki/Topological_sorting
- https://en.wikipedia.org/wiki/Transaction_log

## Storage

- https://thrift.apache.org

## Type System, from Dynamic to Static

- describe how we'd deal with constants like "0", and type number(?)
- explain how we grow a language to be statically typed

# Examples

## Final Sign-Off

We have a complex sheet requiring a formal sign off by an operator

	worksheet requires_review { ...

We track the version at which the operator signed off in `signed_off_version`, and the property of being `signed_off` is computed

	worksheet sign_off {
		1:requires_review requires_review
		2:signed_off_version number(0)
		3:signed_off computed {
			return requires_review.version == signed_off_version
		}
	}

This ensures that any modification to the sheet `requires_review` nullifies the sign off.

## Modeling Declarations

TBD

## Modeling Fees

TBD

### Modeling Assets, Debts, and Income

TBD
