Feature: Show how worksheets testing works

Scenario: our first example
	Given definitions example.ws
	And v = worksheet(simple)
	Then v.name = "Alice"
	Then v
		| name | "Alice" |

Scenario: our second example
	Given definitions example.ws
	And v = worksheet(simple)
		| name | "Alice" |
	Then v.age = 7
	Then v
		| name | "Alice" |
		| age  | 7       |
	Then v.name = "Bob"
	Then v
		| name | "Bob"   |
		| age  | 7       |
