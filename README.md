## Overview

Let's start with a motivating example, easily representing a borrower's legal name, age, and determining if they are of legal age to get a mortgage.

We first describe the domain in a special purpose language

	worksheet borrower {
		1:first_name text
		2:last_name text
		3:dob date
		4:can_take_mortgage computed {
			return dob > now + 18 years
		}
	}

From this definition, we generate Golang hooks to create worksheets, manipulate them (CRUD), and store them

	joey := worksheet.Create("borrower")
	joey.SetText("first_name", "Joey")
	joey.SetText("last_name", "Pizzapie")
	joey.SetDate(1980, 5, 23)

We can query worksheet

	joey.GetBool("can_take_mortgage")

Store it

	bytes, err := joey.Marshal()

Retrieve it from storage

	joey, err := worksheet.Unmarshal("borrower", bytes)

## Worksheets

All centers around the concept of a `worksheet` with named fields which have a predetermined type.

	worksheet person {
		1:age number(0)
		2:first_name text
	}

The general syntax for a field is

	index:name type [extras]

(We explain the need for the index in the storage section.)

### Types

We have only three types

- `number(n)` number with precision of n decimal places
- `text` arbitraty length text
- `map[W]` collection of worksheets `W` indexed by the key of `W`

Specific details about each of these types are given in dedicated sections.

numbers are what we normally of think of them + undefined
text same + undefined
maps is the universe of all maps

### Input Fields

The simplest fields we have are there to store values. In the example above, both `age` and `first_name` are input fields. These can be edited and read freely.

### Constrained Fields

We can also constrain fields

	3:social_security_number number(0) constrained_by {
		100_00_0000 <= social_security_number
		social_security_number <= 999_99_9999
	}

When fields are constrained, edits which do not satisfy the constraint are rejected.

### Computed Fields

- explain, and push syntax of language further down

### Versioning

all worksheets are versionned, it's like if all worksheets have

    version numeric(0)

field on them, all edits to worksheets use optimistic locking

### Enums

While the language does not support enums, these can be described easily with the use of single field worksheet

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

And the Golang hooks allow handling of single field worksheets in a natural way

	borrower.SetText("suffix", "Jr.")

Or

	borrower.GetText("suffix")

## Numbers

Since we intent to eventally have a statically typed langugage, 

### Addition, Substraction

When adding or subtracting numbers, the rule for decimal treatment is

`number(n) op number(m)` yields `number(max(n, m))`

where `op` is either `+` or `-`.

So for instance `5.03` + `6.000` would yield `11.030` as a `number(3)` even though it could be represented as `number(2)`.

### Explicit Rounding When Dividing

e.g. we need to create a map[repayments] which split the cost of taxes, say 700, in 12 (that's 58.3333... if it were evenly split)

say we have numeric(2)
force / operator to be postfixed by rounding mode (always!)

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

## Text

## Maps

We have

	map[W]

which represents a map of worksheets.

### Keyed Worksheets

worksheets which can added into maps must keyed

    keyed_by { fields }

- N-tuples of fields
- keys the worksheet
- required for worksheets used in map[worksheet] types

once a worksheet is in a map, can't do an edit leading to changing a key, otherwhise we'd need to re-hash (and this would cause two worksheets to potentially clash if they're key differed, and are not the same)

if we think of keys as an "index", then we never change an index's value once it is attached to the graph

so process is you create your worksheet, prep the data, assign the key
when you attach it (put it into a map), the key becomes immutable

### operations on maps

- add: can only add if key not present (no implicit replace)
- delete: should we be strict about deletion, i.e. require the key to be present, or not?
- check presence, like `_, ok := the_map[the_key]`
- size: gets the size of the map (note: should we keep len to be closer to go?)
- iterations `for k := range range the_map {` or `for k, v := range the_map`

when iterating over maps, iteration order is the order in which items were added in (essentially a hash map + linked list of the elements)

# other

## constraints

    field_name field_type constrained_by { expression }


## Computational Model

- all values are optional

- operations take into account optionality
  e.g. undefined + 6 yields undefined

- no cycles in function graphs, no recursion, etc. but must be without cycles (outputs from inputs)

## Edits

- can set a field (or set it to undefined, which really unsets)
- for maps
-- can add <<
-- can remove delete the_map[the_key]

want to be able to build progressive diffs before saving them (e.g. to support 'draft mode' when editing the application)

## Security

### ACLs
### Encyption
### Audit Trail

## Storage

- of the worksheets
- of edits (i.e. the redo log)
- marshalling: models can be saved in same format as thrift (hence, need index marker for each values)

## Introspection

- query an input and get concrete AST of how it is calculated from all raw values
- query an input to see every value it flows into, i.e. all computed fields using this input

## Implementation

### Dynamically Typed To Start

- describe how we'd deal with constants like "0", and type number(?)

### Towards Statically Typed

- explain how we grow a language to be statically typed

ideas from

- https://en.wikipedia.org/wiki/Rete_algorithm
- https://en.wikipedia.org/wiki/Topological_sorting
- https://en.wikipedia.org/wiki/Transaction_log
- https://thrift.apache.org
