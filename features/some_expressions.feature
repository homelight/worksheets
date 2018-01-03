Feature: Fun with simple expressions

Scenario: put stuff in num
	Given definitions some_expressions.ws
	And v = worksheet(some_expressions)
	Then v.num = 3
	Then v
		| num                            |   3      |
		| num_plus_two                   |   5.00   |
		| num_more_decimals              |   3.0000 |
		| volume_of_sphere_of_num_radius | 113.0973 |
