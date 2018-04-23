type example worksheet {
	1:nums []number[0]
	2:sum  number[0] computed_by {
		return sum(nums)
	}
}
