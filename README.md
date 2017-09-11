# Overview

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

All worksheets are versionned, with their version number starting a `1`. In a way, all worksheets have a field

	version numeric(0)

Which is set to `1` upon creation, and incremented on every edit. Versions are used to detect concurrent edits, and abort an edit that was done on an older version of the worksheet than the one it would now be apply to. Edits are discussed in greater detail later.

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

TBD

## Keyed Worksheets, Maps, and Tuples

In addition to the structures covered earlier, we have

	map[W]

which represents a collection of worksheets `W`, indexed by the key of `W`.

### Tuples

As we shall see in the next section, worksheets' keys can be one or multiple values joined together in N-tuples. To represent this, we introduce

	tuple[T1, ..., Tn]

NOTE: We don't want to allow `tuple[map[foo]]`, so how do we differentiate the simple types, from more complex types like maps? Maps can be the type of fields, but the type which can go in tuples is a subset of that. Maybe 'base type' as `S` and field type `T` is a sufficient distinction? Then maps and tuples would be field types, one over worksheets, the other over only base types.

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

- can set a field (or set it to undefined, which really unsets)
- for maps
-- can add <<
-- can remove delete the_map[the_key]

want to be able to build progressive diffs before saving them (e.g. to support 'draft mode' when editing the application)

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

## Modeling Declarations

TBD

## Modeling Fees

TBD

### Modeling Assets, Debts, and Income

TBD
