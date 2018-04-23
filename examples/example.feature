Feature: An example of how to use feature based tests

Background:
	Given load "example.ws"

Scenario: sum starts at 0
	When create ws "example"
	Then assert ws.sum 0

Scenario: sum adds numbers nums
	When create ws "example"
	And append ws.nums 2
	Then assert ws.sum 2

	When append ws.nums 3
	Then assert ws.sum 5

	When append ws.nums
		|  5 |
		|  8 |
		| 13 |
	Then assert ws
		| sum | 31 |
		| -   |    |
