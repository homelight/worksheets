type player enum {
	round,
	cross,
}

type tic_tac_toe worksheet {
	// grid (rows: 0, 1, 2; columns: a, b, c)
	1:cell_a_0 player
	2:cell_b_0 player
	3:cell_c_0 player
	4:cell_a_1 player
	5:cell_b_1 player
	6:cell_c_1 player
	7:cell_a_2 player
	8:cell_b_2 player
	9:cell_c_2 player

	// winnings by rows
	10: has_won_0 player computed_by {
		return if(cell_b_0 == cell_a_0 && cell_c_0 == cell_a_0, cell_a_0)
	}
	11: has_won_1 player computed_by {
		return if(cell_b_1 == cell_a_1 && cell_c_1 == cell_a_1, cell_a_1)
	}
	12: has_won_2 player computed_by {
		return if(cell_b_2 == cell_a_2 && cell_c_2 == cell_a_0, cell_a_2)
	}
	13: has_won_by_row player computed_by {
		return first_of(
			has_won_0,
			has_won_1,
			has_won_2,
		)
	}

	// winnings by columns
	20: has_won_a player computed_by {
		return if(cell_a_0 == cell_a_1 && cell_a_0 == cell_a_2, cell_a_0)
	}
	21: has_won_b player computed_by {
		return if(cell_b_0 == cell_b_1 && cell_b_0 == cell_b_2, cell_b_0)
	}
	22: has_won_c player computed_by {
		return if(cell_c_0 == cell_c_1 && cell_c_0 == cell_c_2, cell_c_0)
	}
	23: has_won_by_column player computed_by {
		return first_of(
			has_won_a,
			has_won_b,
			has_won_c,
		)
	}

	// winnings by diagonal
	31: has_won_diag1 player computed_by {
		return if(cell_a_0 == cell_b_1 && cell_a_0 == cell_c_2, cell_a_0)
	}
	32: has_won_diag2 player computed_by {
		return if(cell_a_2 == cell_b_1 && cell_a_2 == cell_c_0, cell_a_2)
	}
	33: has_won_by_diagonal player computed_by {
		return first_of(
			has_won_diag1,
			has_won_diag2,
		)
	}

	// and the winner is
	43: winner player computed_by {
		return first_of(
			has_won_by_row,
			has_won_by_column,
			has_won_by_diagonal,
		)
	}
}
