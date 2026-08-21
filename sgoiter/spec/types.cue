package spec

#CType: {
	name:      string
	size:      int
	is_signed: bool
	is_scalar: bool
	go_type:   string
}

types: [...#CType] & [
	// 8 bits
	{name: "char", size: 1, is_signed: true, is_scalar: true, go_type: "int8"},
	{name: "signed char", size: 1, is_signed: true, is_scalar: true, go_type: "int8"},
	{name: "int8_t", size: 1, is_signed: true, is_scalar: true, go_type: "int8"},
	{name: "i8", size: 1, is_signed: true, is_scalar: true, go_type: "int8"},
	{name: "unsigned char", size: 1, is_signed: false, is_scalar: true, go_type: "byte"},
	{name: "uint8_t", size: 1, is_signed: false, is_scalar: true, go_type: "byte"},
	{name: "u8", size: 1, is_signed: false, is_scalar: true, go_type: "byte"},
	{name: "bool", size: 1, is_signed: false, is_scalar: true, go_type: "bool"},
	{name: "_Bool", size: 1, is_signed: false, is_scalar: true, go_type: "bool"},

	// 16 bits
	{name: "short", size: 2, is_signed: true, is_scalar: true, go_type: "int16"},
	{name: "signed short", size: 2, is_signed: true, is_scalar: true, go_type: "int16"},
	{name: "int16_t", size: 2, is_signed: true, is_scalar: true, go_type: "int16"},
	{name: "i16", size: 2, is_signed: true, is_scalar: true, go_type: "int16"},
	{name: "unsigned short", size: 2, is_signed: false, is_scalar: true, go_type: "uint16"},
	{name: "uint16_t", size: 2, is_signed: false, is_scalar: true, go_type: "uint16"},
	{name: "u16", size: 2, is_signed: false, is_scalar: true, go_type: "uint16"},

	// 32 bits
	{name: "int", size: 4, is_signed: true, is_scalar: true, go_type: "int"},
	{name: "signed int", size: 4, is_signed: true, is_scalar: true, go_type: "int32"},
	{name: "int32_t", size: 4, is_signed: true, is_scalar: true, go_type: "int32"},
	{name: "i32", size: 4, is_signed: true, is_scalar: true, go_type: "int32"},
	{name: "unsigned int", size: 4, is_signed: false, is_scalar: true, go_type: "uint32"},
	{name: "unsigned", size: 4, is_signed: false, is_scalar: true, go_type: "uint32"},
	{name: "uint32_t", size: 4, is_signed: false, is_scalar: true, go_type: "uint32"},
	{name: "u32", size: 4, is_signed: false, is_scalar: true, go_type: "uint32"},
	{name: "float", size: 4, is_signed: true, is_scalar: true, go_type: "float32"},

	// 64 bits
	{name: "long", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "signed long", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "long long", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "signed long long", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "int64_t", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "i64", size: 8, is_signed: true, is_scalar: true, go_type: "int64"},
	{name: "unsigned long", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "unsigned long long", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "uint64_t", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "u64", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "size_t", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "usize", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "uintptr_t", size: 8, is_signed: false, is_scalar: true, go_type: "uint64"},
	{name: "double", size: 8, is_signed: true, is_scalar: true, go_type: "float64"},

	// 128 bits
	{name: "uint128_t", size: 16, is_signed: false, is_scalar: false, go_type: "[2]uint64"},
	{name: "int128_t", size: 16, is_signed: true, is_scalar: false, go_type: "[2]uint64"},
]
