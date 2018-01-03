worksheet some_expressions {
	1:num number[2]
	2:num_plus_two number[2] computed_by {
		num + 2 round down 2
	}
	3:num_more_decimals number[4] computed_by {
		num round down 4
	}
	4:volume_of_sphere_of_num_radius number[4] computed_by {
		(4 / 3 round down 6 * 3.141593 * num * num * num) round down 4
	}
}
