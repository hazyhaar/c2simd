package spec

#RewriteRule: {
	name:          string
	description:   string
	match_pattern: string
	replacement:   string
}

rules: [...#RewriteRule] & [
	{
		name:          "RotateLeft"
		description:   "Rewrite left rotation"
		match_pattern: "rotl"
		replacement:   "bits.RotateLeft"
	},
	{
		name:          "BooleanEncapsulation"
		description:   "Encapsulate boolean condition"
		match_pattern: "cond ? a : b"
		replacement:   "if cond { a } else { b }"
	},
]
