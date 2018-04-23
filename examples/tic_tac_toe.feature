Feature: Tic Tac Toe

Background:
	Given load "tic_tac_toe.ws"
	And create ws "tic_tac_toe"

Scenario: column a
	Then set ws
		| cell_a_0          | "round" |
		| cell_a_1          | "round" |
		| cell_a_2          | "round" |
	Then assert ws
		| cell_a_0          | "round" |
		| cell_a_1          | "round" |
		| cell_a_2          | "round" |
		| has_won_a         | "round" |
		| has_won_by_column | "round" |
		| winner            | "round" |

Scenario: row 1
	Then set ws
		| cell_a_1          | "cross" |
		| cell_b_1          | "cross" |
		| cell_c_1          | "cross" |
	Then assert ws
		| cell_a_1          | "cross" |
		| cell_b_1          | "cross" |
		| cell_c_1          | "cross" |
		| has_won_1         | "cross" |
		| has_won_by_row    | "cross" |
		| winner            | "cross" |

Scenario: diagonal
	Then set ws
		| cell_a_0          | "cross" |
		| cell_b_1          | "cross" |
		| cell_c_2          | "cross" |
	Then assert ws
		| has_won_diag1       | "cross" |
		| has_won_by_diagonal | "cross" |
		| winner              | "cross" |
		| -                   |         |
