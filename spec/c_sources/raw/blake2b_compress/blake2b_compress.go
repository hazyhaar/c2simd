package raw_blake2b_compress



import (
	"math"
	"reflect"
	"unsafe"

	"modernc.org/libc"
)

var _ = math.Pi
var _ reflect.Type
var _ unsafe.Pointer

const INT16_MAX = 0x7fff
const INT32_MAX = 0x7fffffff
const INT64_MAX = 0x7fffffffffffffff
const INT8_MAX = 0x7f
const INTMAX_MAX = "INT64_MAX"
const INTMAX_MIN = "INT64_MIN"
const INTPTR_MAX = "INT64_MAX"
const INTPTR_MIN = "INT64_MIN"
const INT_FAST16_MAX = "INT32_MAX"
const INT_FAST16_MIN = "INT32_MIN"
const INT_FAST32_MAX = "INT32_MAX"
const INT_FAST32_MIN = "INT32_MIN"
const INT_FAST64_MAX = "INT64_MAX"
const INT_FAST64_MIN = "INT64_MIN"
const INT_FAST8_MAX = "INT8_MAX"
const INT_FAST8_MIN = "INT8_MIN"
const INT_LEAST16_MAX = "INT16_MAX"
const INT_LEAST16_MIN = "INT16_MIN"
const INT_LEAST32_MAX = "INT32_MAX"
const INT_LEAST32_MIN = "INT32_MIN"
const INT_LEAST64_MAX = "INT64_MAX"
const INT_LEAST64_MIN = "INT64_MIN"
const INT_LEAST8_MAX = "INT8_MAX"
const INT_LEAST8_MIN = "INT8_MIN"
const PTRDIFF_MAX = "INT64_MAX"
const PTRDIFF_MIN = "INT64_MIN"
const SIG_ATOMIC_MAX = "INT32_MAX"
const SIG_ATOMIC_MIN = "INT32_MIN"
const SIZE_MAX = "UINT64_MAX"
const UINT16_MAX = 0xffff
const UINT32_MAX = "0xffffffffu"
const UINT64_MAX = "0xffffffffffffffffu"
const UINT8_MAX = 0xff
const UINTMAX_MAX = "UINT64_MAX"
const UINTPTR_MAX = "UINT64_MAX"
const UINT_FAST16_MAX = "UINT32_MAX"
const UINT_FAST32_MAX = "UINT32_MAX"
const UINT_FAST64_MAX = "UINT64_MAX"
const UINT_FAST8_MAX = "UINT8_MAX"
const UINT_LEAST16_MAX = "UINT16_MAX"
const UINT_LEAST32_MAX = "UINT32_MAX"
const UINT_LEAST64_MAX = "UINT64_MAX"
const UINT_LEAST8_MAX = "UINT8_MAX"
const WINT_MAX = "UINT32_MAX"
const WINT_MIN = 0
const _GNU_SOURCE = 1
const _LP64 = 1
const _STDC_PREDEF_H = 1
const __ATOMIC_ACQUIRE = 2
const __ATOMIC_ACQ_REL = 4
const __ATOMIC_CONSUME = 1
const __ATOMIC_HLE_ACQUIRE = 65536
const __ATOMIC_HLE_RELEASE = 131072
const __ATOMIC_RELAXED = 0
const __ATOMIC_RELEASE = 3
const __ATOMIC_SEQ_CST = 5
const __BFLT16_DECIMAL_DIG__ = 4
const __BFLT16_DENORM_MIN__ = "9.18354961579912115600575419704879436e-41B"
const __BFLT16_DIG__ = 2
const __BFLT16_EPSILON__ = "7.81250000000000000000000000000000000e-3B"
const __BFLT16_HAS_DENORM__ = 1
const __BFLT16_HAS_INFINITY__ = 1
const __BFLT16_HAS_QUIET_NAN__ = 1
const __BFLT16_IS_IEC_60559__ = 0
const __BFLT16_MANT_DIG__ = 8
const __BFLT16_MAX_10_EXP__ = 38
const __BFLT16_MAX_EXP__ = 128
const __BFLT16_MAX__ = "3.38953138925153547590470800371487867e+38B"
const __BFLT16_MIN__ = "1.17549435082228750796873653722224568e-38B"
const __BFLT16_NORM_MAX__ = "3.38953138925153547590470800371487867e+38B"
const __BIGGEST_ALIGNMENT__ = 16
const __BIG_ENDIAN = 4321
const __BYTE_ORDER = 1234
const __BYTE_ORDER__ = "__ORDER_LITTLE_ENDIAN__"
const __CCGO__ = 1
const __CET__ = 3
const __CHAR_BIT__ = 8
const __DBL_DECIMAL_DIG__ = 17
const __DBL_DIG__ = 15
const __DBL_HAS_DENORM__ = 1
const __DBL_HAS_INFINITY__ = 1
const __DBL_HAS_QUIET_NAN__ = 1
const __DBL_IS_IEC_60559__ = 1
const __DBL_MANT_DIG__ = 53
const __DBL_MAX_10_EXP__ = 308
const __DBL_MAX_EXP__ = 1024
const __DEC128_EPSILON__ = 1e-33
const __DEC128_MANT_DIG__ = 34
const __DEC128_MAX_EXP__ = 6145
const __DEC128_MAX__ = "9.999999999999999999999999999999999E6144"
const __DEC128_MIN__ = 1e-6143
const __DEC128_SUBNORMAL_MIN__ = 0.000000000000000000000000000000001e-6143
const __DEC32_EPSILON__ = 1e-6
const __DEC32_MANT_DIG__ = 7
const __DEC32_MAX_EXP__ = 97
const __DEC32_MAX__ = 9.999999e96
const __DEC32_MIN__ = 1e-95
const __DEC32_SUBNORMAL_MIN__ = 0.000001e-95
const __DEC64_EPSILON__ = 1e-15
const __DEC64_MANT_DIG__ = 16
const __DEC64_MAX_EXP__ = 385
const __DEC64_MAX__ = "9.999999999999999E384"
const __DEC64_MIN__ = 1e-383
const __DEC64_SUBNORMAL_MIN__ = 0.000000000000001e-383
const __DECIMAL_BID_FORMAT__ = 1
const __DECIMAL_DIG__ = 17
const __DEC_EVAL_METHOD__ = 2
const __ELF__ = 1
const __FINITE_MATH_ONLY__ = 0
const __FLOAT_WORD_ORDER__ = "__ORDER_LITTLE_ENDIAN__"
const __FLT128_DECIMAL_DIG__ = 36
const __FLT128_DENORM_MIN__ = 6.47517511943802511092443895822764655e-4966
const __FLT128_DIG__ = 33
const __FLT128_EPSILON__ = 1.92592994438723585305597794258492732e-34
const __FLT128_HAS_DENORM__ = 1
const __FLT128_HAS_INFINITY__ = 1
const __FLT128_HAS_QUIET_NAN__ = 1
const __FLT128_IS_IEC_60559__ = 1
const __FLT128_MANT_DIG__ = 113
const __FLT128_MAX_10_EXP__ = 4932
const __FLT128_MAX_EXP__ = 16384
const __FLT128_MAX__ = "1.18973149535723176508575932662800702e+4932"
const __FLT128_MIN__ = 3.36210314311209350626267781732175260e-4932
const __FLT128_NORM_MAX__ = "1.18973149535723176508575932662800702e+4932"
const __FLT16_DECIMAL_DIG__ = 5
const __FLT16_DENORM_MIN__ = 5.96046447753906250000000000000000000e-8
const __FLT16_DIG__ = 3
const __FLT16_EPSILON__ = 9.76562500000000000000000000000000000e-4
const __FLT16_HAS_DENORM__ = 1
const __FLT16_HAS_INFINITY__ = 1
const __FLT16_HAS_QUIET_NAN__ = 1
const __FLT16_IS_IEC_60559__ = 1
const __FLT16_MANT_DIG__ = 11
const __FLT16_MAX_10_EXP__ = 4
const __FLT16_MAX_EXP__ = 16
const __FLT16_MAX__ = 6.55040000000000000000000000000000000e+4
const __FLT16_MIN__ = 6.10351562500000000000000000000000000e-5
const __FLT16_NORM_MAX__ = 6.55040000000000000000000000000000000e+4
const __FLT32X_DECIMAL_DIG__ = 17
const __FLT32X_DENORM_MIN__ = 4.94065645841246544176568792868221372e-324
const __FLT32X_DIG__ = 15
const __FLT32X_EPSILON__ = 2.22044604925031308084726333618164062e-16
const __FLT32X_HAS_DENORM__ = 1
const __FLT32X_HAS_INFINITY__ = 1
const __FLT32X_HAS_QUIET_NAN__ = 1
const __FLT32X_IS_IEC_60559__ = 1
const __FLT32X_MANT_DIG__ = 53
const __FLT32X_MAX_10_EXP__ = 308
const __FLT32X_MAX_EXP__ = 1024
const __FLT32X_MAX__ = 1.79769313486231570814527423731704357e+308
const __FLT32X_MIN__ = 2.22507385850720138309023271733240406e-308
const __FLT32X_NORM_MAX__ = 1.79769313486231570814527423731704357e+308
const __FLT32_DECIMAL_DIG__ = 9
const __FLT32_DENORM_MIN__ = 1.40129846432481707092372958328991613e-45
const __FLT32_DIG__ = 6
const __FLT32_EPSILON__ = 1.19209289550781250000000000000000000e-7
const __FLT32_HAS_DENORM__ = 1
const __FLT32_HAS_INFINITY__ = 1
const __FLT32_HAS_QUIET_NAN__ = 1
const __FLT32_IS_IEC_60559__ = 1
const __FLT32_MANT_DIG__ = 24
const __FLT32_MAX_10_EXP__ = 38
const __FLT32_MAX_EXP__ = 128
const __FLT32_MAX__ = 3.40282346638528859811704183484516925e+38
const __FLT32_MIN__ = 1.17549435082228750796873653722224568e-38
const __FLT32_NORM_MAX__ = 3.40282346638528859811704183484516925e+38
const __FLT64X_DECIMAL_DIG__ = 36
const __FLT64X_DENORM_MIN__ = 6.47517511943802511092443895822764655e-4966
const __FLT64X_DIG__ = 33
const __FLT64X_EPSILON__ = 1.92592994438723585305597794258492732e-34
const __FLT64X_HAS_DENORM__ = 1
const __FLT64X_HAS_INFINITY__ = 1
const __FLT64X_HAS_QUIET_NAN__ = 1
const __FLT64X_IS_IEC_60559__ = 1
const __FLT64X_MANT_DIG__ = 113
const __FLT64X_MAX_10_EXP__ = 4932
const __FLT64X_MAX_EXP__ = 16384
const __FLT64X_MAX__ = "1.18973149535723176508575932662800702e+4932"
const __FLT64X_MIN__ = 3.36210314311209350626267781732175260e-4932
const __FLT64X_NORM_MAX__ = "1.18973149535723176508575932662800702e+4932"
const __FLT64_DECIMAL_DIG__ = 17
const __FLT64_DENORM_MIN__ = 4.94065645841246544176568792868221372e-324
const __FLT64_DIG__ = 15
const __FLT64_EPSILON__ = 2.22044604925031308084726333618164062e-16
const __FLT64_HAS_DENORM__ = 1
const __FLT64_HAS_INFINITY__ = 1
const __FLT64_HAS_QUIET_NAN__ = 1
const __FLT64_IS_IEC_60559__ = 1
const __FLT64_MANT_DIG__ = 53
const __FLT64_MAX_10_EXP__ = 308
const __FLT64_MAX_EXP__ = 1024
const __FLT64_MAX__ = 1.79769313486231570814527423731704357e+308
const __FLT64_MIN__ = 2.22507385850720138309023271733240406e-308
const __FLT64_NORM_MAX__ = 1.79769313486231570814527423731704357e+308
const __FLT_DECIMAL_DIG__ = 9
const __FLT_DENORM_MIN__ = 1.40129846432481707092372958328991613e-45
const __FLT_DIG__ = 6
const __FLT_EPSILON__ = 1.19209289550781250000000000000000000e-7
const __FLT_EVAL_METHOD_TS_18661_3__ = 0
const __FLT_EVAL_METHOD__ = 0
const __FLT_HAS_DENORM__ = 1
const __FLT_HAS_INFINITY__ = 1
const __FLT_HAS_QUIET_NAN__ = 1
const __FLT_IS_IEC_60559__ = 1
const __FLT_MANT_DIG__ = 24
const __FLT_MAX_10_EXP__ = 38
const __FLT_MAX_EXP__ = 128
const __FLT_MAX__ = 3.40282346638528859811704183484516925e+38
const __FLT_MIN__ = 1.17549435082228750796873653722224568e-38
const __FLT_NORM_MAX__ = 3.40282346638528859811704183484516925e+38
const __FLT_RADIX__ = 2
const __FUNCTION__ = "__func__"
const __FXSR__ = 1
const __GCC_ASM_FLAG_OUTPUTS__ = 1
const __GCC_ATOMIC_BOOL_LOCK_FREE = 2
const __GCC_ATOMIC_CHAR16_T_LOCK_FREE = 2
const __GCC_ATOMIC_CHAR32_T_LOCK_FREE = 2
const __GCC_ATOMIC_CHAR_LOCK_FREE = 2
const __GCC_ATOMIC_INT_LOCK_FREE = 2
const __GCC_ATOMIC_LLONG_LOCK_FREE = 2
const __GCC_ATOMIC_LONG_LOCK_FREE = 2
const __GCC_ATOMIC_POINTER_LOCK_FREE = 2
const __GCC_ATOMIC_SHORT_LOCK_FREE = 2
const __GCC_ATOMIC_TEST_AND_SET_TRUEVAL = 1
const __GCC_ATOMIC_WCHAR_T_LOCK_FREE = 2
const __GCC_CONSTRUCTIVE_SIZE = 64
const __GCC_DESTRUCTIVE_SIZE = 64
const __GCC_HAVE_DWARF2_CFI_ASM = 1
const __GCC_HAVE_SYNC_COMPARE_AND_SWAP_1 = 1
const __GCC_HAVE_SYNC_COMPARE_AND_SWAP_2 = 1
const __GCC_HAVE_SYNC_COMPARE_AND_SWAP_4 = 1
const __GCC_HAVE_SYNC_COMPARE_AND_SWAP_8 = 1
const __GCC_IEC_559 = 2
const __GCC_IEC_559_COMPLEX = 2
const __GNUC_EXECUTION_CHARSET_NAME = "UTF-8"
const __GNUC_MINOR__ = 3
const __GNUC_PATCHLEVEL__ = 0
const __GNUC_STDC_INLINE__ = 1
const __GNUC_WIDE_EXECUTION_CHARSET_NAME = "UTF-32LE"
const __GNUC__ = 13
const __GXX_ABI_VERSION = 1018
const __HAVE_SPECULATION_SAFE_VALUE = 1
const __INT16_MAX__ = 0x7fff
const __INT32_MAX__ = 0x7fffffff
const __INT32_TYPE__ = "int"
const __INT64_MAX__ = 0x7fffffffffffffff
const __INT8_MAX__ = 0x7f
const __INTMAX_MAX__ = 0x7fffffffffffffff
const __INTMAX_WIDTH__ = 64
const __INTPTR_MAX__ = 0x7fffffffffffffff
const __INTPTR_WIDTH__ = 64
const __INT_FAST16_MAX__ = 0x7fffffffffffffff
const __INT_FAST16_WIDTH__ = 64
const __INT_FAST32_MAX__ = 0x7fffffffffffffff
const __INT_FAST32_WIDTH__ = 64
const __INT_FAST64_MAX__ = 0x7fffffffffffffff
const __INT_FAST64_WIDTH__ = 64
const __INT_FAST8_MAX__ = 0x7f
const __INT_FAST8_WIDTH__ = 8
const __INT_LEAST16_MAX__ = 0x7fff
const __INT_LEAST16_WIDTH__ = 16
const __INT_LEAST32_MAX__ = 0x7fffffff
const __INT_LEAST32_TYPE__ = "int"
const __INT_LEAST32_WIDTH__ = 32
const __INT_LEAST64_MAX__ = 0x7fffffffffffffff
const __INT_LEAST64_WIDTH__ = 64
const __INT_LEAST8_MAX__ = 0x7f
const __INT_LEAST8_WIDTH__ = 8
const __INT_MAX__ = 0x7fffffff
const __INT_WIDTH__ = 32
const __LDBL_DECIMAL_DIG__ = 17
const __LDBL_DENORM_MIN__ = 4.94065645841246544176568792868221372e-324
const __LDBL_DIG__ = 15
const __LDBL_EPSILON__ = 2.22044604925031308084726333618164062e-16
const __LDBL_HAS_DENORM__ = 1
const __LDBL_HAS_INFINITY__ = 1
const __LDBL_HAS_QUIET_NAN__ = 1
const __LDBL_IS_IEC_60559__ = 1
const __LDBL_MANT_DIG__ = 53
const __LDBL_MAX_10_EXP__ = 308
const __LDBL_MAX_EXP__ = 1024
const __LDBL_MAX__ = 1.79769313486231570814527423731704357e+308
const __LDBL_MIN__ = 2.22507385850720138309023271733240406e-308
const __LDBL_NORM_MAX__ = 1.79769313486231570814527423731704357e+308
const __LITTLE_ENDIAN = 1234
const __LONG_DOUBLE_64__ = 1
const __LONG_LONG_MAX__ = 0x7fffffffffffffff
const __LONG_LONG_WIDTH__ = 64
const __LONG_MAX = 0x7fffffffffffffff
const __LONG_MAX__ = 0x7fffffffffffffff
const __LONG_WIDTH__ = 64
const __LP64__ = 1
const __MMX_WITH_SSE__ = 1
const __MMX__ = 1
const __NO_INLINE__ = 1
const __ORDER_BIG_ENDIAN__ = 4321
const __ORDER_LITTLE_ENDIAN__ = 1234
const __ORDER_PDP_ENDIAN__ = 3412
const __PIC__ = 2
const __PIE__ = 2
const __PRAGMA_REDEFINE_EXTNAME = 1
const __PRETTY_FUNCTION__ = "__func__"
const __PTRDIFF_MAX__ = 0x7fffffffffffffff
const __PTRDIFF_WIDTH__ = 64
const __SCHAR_MAX__ = 0x7f
const __SCHAR_WIDTH__ = 8
const __SEG_FS = 1
const __SEG_GS = 1
const __SHRT_MAX__ = 0x7fff
const __SHRT_WIDTH__ = 16
const __SIG_ATOMIC_MAX__ = 0x7fffffff
const __SIG_ATOMIC_TYPE__ = "int"
const __SIG_ATOMIC_WIDTH__ = 32
const __SIZEOF_DOUBLE__ = 8
const __SIZEOF_FLOAT128__ = 16
const __SIZEOF_FLOAT80__ = 16
const __SIZEOF_FLOAT__ = 4
const __SIZEOF_INT128__ = 16
const __SIZEOF_INT__ = 4
const __SIZEOF_LONG_DOUBLE__ = 8
const __SIZEOF_LONG_LONG__ = 8
const __SIZEOF_LONG__ = 8
const __SIZEOF_POINTER__ = 8
const __SIZEOF_PTRDIFF_T__ = 8
const __SIZEOF_SHORT__ = 2
const __SIZEOF_SIZE_T__ = 8
const __SIZEOF_WCHAR_T__ = 4
const __SIZEOF_WINT_T__ = 4
const __SIZE_MAX__ = 0xffffffffffffffff
const __SIZE_WIDTH__ = 64
const __SSE2_MATH__ = 1
const __SSE2__ = 1
const __SSE_MATH__ = 1
const __SSE__ = 1
const __SSP_STRONG__ = 3
const __STDC_HOSTED__ = 1
const __STDC_IEC_559_COMPLEX__ = 1
const __STDC_IEC_559__ = 1
const __STDC_IEC_60559_BFP__ = 201404
const __STDC_IEC_60559_COMPLEX__ = 201404
const __STDC_ISO_10646__ = 201706
const __STDC_UTF_16__ = 1
const __STDC_UTF_32__ = 1
const __STDC_VERSION__ = 201710
const __STDC__ = 1
const __UINT16_MAX__ = 0xffff
const __UINT32_MAX__ = 0xffffffff
const __UINT64_MAX__ = 0xffffffffffffffff
const __UINT8_MAX__ = 0xff
const __UINTMAX_MAX__ = 0xffffffffffffffff
const __UINTPTR_MAX__ = 0xffffffffffffffff
const __UINT_FAST16_MAX__ = 0xffffffffffffffff
const __UINT_FAST32_MAX__ = 0xffffffffffffffff
const __UINT_FAST64_MAX__ = 0xffffffffffffffff
const __UINT_FAST8_MAX__ = 0xff
const __UINT_LEAST16_MAX__ = 0xffff
const __UINT_LEAST32_MAX__ = 0xffffffff
const __UINT_LEAST64_MAX__ = 0xffffffffffffffff
const __UINT_LEAST8_MAX__ = 0xff
const __USE_TIME_BITS64 = 1
const __VERSION__ = "13.3.0"
const __WCHAR_MAX__ = 0x7fffffff
const __WCHAR_TYPE__ = "int"
const __WCHAR_WIDTH__ = 32
const __WINT_MAX__ = 0xffffffff
const __WINT_MIN__ = 0
const __WINT_WIDTH__ = 32
const __amd64 = 1
const __amd64__ = 1
const __code_model_small__ = 1
const __gnu_linux__ = 1
const __k8 = 1
const __k8__ = 1
const __linux = 1
const __linux__ = 1
const __pic__ = 2
const __pie__ = 2
const __restrict_arr = "restrict"
const __unix = 1
const __unix__ = 1
const __x86_64 = 1
const __x86_64__ = 1
const linux = 1
const unix = 1

type __builtin_va_list = uintptr

type __predefined_size_t = uint64

type __predefined_wchar_t = int32

type __predefined_ptrdiff_t = int64

type uintptr_t = uint64

type intptr_t = int64

type int8_t = int8

type int16_t = int16

type int32_t = int32

type int64_t = int64

type intmax_t = int64

type uint8_t = uint8

type uint16_t = uint16

type uint32_t = uint32

type uint64_t = uint64

type uintmax_t = uint64

type int_fast8_t = int8

type int_fast64_t = int64

type int_least8_t = int8

type int_least16_t = int16

type int_least32_t = int32

type int_least64_t = int64

type uint_fast8_t = uint8

type uint_fast64_t = uint64

type uint_least8_t = uint8

type uint_least16_t = uint16

type uint_least32_t = uint32

type uint_least64_t = uint64

type int_fast16_t = int32

type int_fast32_t = int32

type uint_fast16_t = uint32

type uint_fast32_t = uint32

type wchar_t = int32

type max_align_t = struct {
	F__ll int64
	F__ld float64
}

type size_t = uint64

type ptrdiff_t = int64

var blake2b_sigma = [12][16]uint8_t{
	0: {
		1:  uint8(1),
		2:  uint8(2),
		3:  uint8(3),
		4:  uint8(4),
		5:  uint8(5),
		6:  uint8(6),
		7:  uint8(7),
		8:  uint8(8),
		9:  uint8(9),
		10: uint8(10),
		11: uint8(11),
		12: uint8(12),
		13: uint8(13),
		14: uint8(14),
		15: uint8(15),
	},
	1: {
		0:  uint8(14),
		1:  uint8(10),
		2:  uint8(4),
		3:  uint8(8),
		4:  uint8(9),
		5:  uint8(15),
		6:  uint8(13),
		7:  uint8(6),
		8:  uint8(1),
		9:  uint8(12),
		11: uint8(2),
		12: uint8(11),
		13: uint8(7),
		14: uint8(5),
		15: uint8(3),
	},
	2: {
		0:  uint8(11),
		1:  uint8(8),
		2:  uint8(12),
		4:  uint8(5),
		5:  uint8(2),
		6:  uint8(15),
		7:  uint8(13),
		8:  uint8(10),
		9:  uint8(14),
		10: uint8(3),
		11: uint8(6),
		12: uint8(7),
		13: uint8(1),
		14: uint8(9),
		15: uint8(4),
	},
	3: {
		0:  uint8(7),
		1:  uint8(9),
		2:  uint8(3),
		3:  uint8(1),
		4:  uint8(13),
		5:  uint8(12),
		6:  uint8(11),
		7:  uint8(14),
		8:  uint8(2),
		9:  uint8(6),
		10: uint8(5),
		11: uint8(10),
		12: uint8(4),
		14: uint8(15),
		15: uint8(8),
	},
	4: {
		0:  uint8(9),
		2:  uint8(5),
		3:  uint8(7),
		4:  uint8(2),
		5:  uint8(4),
		6:  uint8(10),
		7:  uint8(15),
		8:  uint8(14),
		9:  uint8(1),
		10: uint8(11),
		11: uint8(12),
		12: uint8(6),
		13: uint8(8),
		14: uint8(3),
		15: uint8(13),
	},
	5: {
		0:  uint8(2),
		1:  uint8(12),
		2:  uint8(6),
		3:  uint8(10),
		5:  uint8(11),
		6:  uint8(8),
		7:  uint8(3),
		8:  uint8(4),
		9:  uint8(13),
		10: uint8(7),
		11: uint8(5),
		12: uint8(15),
		13: uint8(14),
		14: uint8(1),
		15: uint8(9),
	},
	6: {
		0:  uint8(12),
		1:  uint8(5),
		2:  uint8(1),
		3:  uint8(15),
		4:  uint8(14),
		5:  uint8(13),
		6:  uint8(4),
		7:  uint8(10),
		9:  uint8(7),
		10: uint8(6),
		11: uint8(3),
		12: uint8(9),
		13: uint8(2),
		14: uint8(8),
		15: uint8(11),
	},
	7: {
		0:  uint8(13),
		1:  uint8(11),
		2:  uint8(7),
		3:  uint8(14),
		4:  uint8(12),
		5:  uint8(1),
		6:  uint8(3),
		7:  uint8(9),
		8:  uint8(5),
		10: uint8(15),
		11: uint8(4),
		12: uint8(8),
		13: uint8(6),
		14: uint8(2),
		15: uint8(10),
	},
	8: {
		0:  uint8(6),
		1:  uint8(15),
		2:  uint8(14),
		3:  uint8(9),
		4:  uint8(11),
		5:  uint8(3),
		7:  uint8(8),
		8:  uint8(12),
		9:  uint8(2),
		10: uint8(13),
		11: uint8(7),
		12: uint8(1),
		13: uint8(4),
		14: uint8(10),
		15: uint8(5),
	},
	9: {
		0:  uint8(10),
		1:  uint8(2),
		2:  uint8(8),
		3:  uint8(4),
		4:  uint8(7),
		5:  uint8(6),
		6:  uint8(1),
		7:  uint8(5),
		8:  uint8(15),
		9:  uint8(11),
		10: uint8(9),
		11: uint8(14),
		12: uint8(3),
		13: uint8(12),
		14: uint8(13),
	},
	10: {
		1:  uint8(1),
		2:  uint8(2),
		3:  uint8(3),
		4:  uint8(4),
		5:  uint8(5),
		6:  uint8(6),
		7:  uint8(7),
		8:  uint8(8),
		9:  uint8(9),
		10: uint8(10),
		11: uint8(11),
		12: uint8(12),
		13: uint8(13),
		14: uint8(14),
		15: uint8(15),
	},
	11: {
		0:  uint8(14),
		1:  uint8(10),
		2:  uint8(4),
		3:  uint8(8),
		4:  uint8(9),
		5:  uint8(15),
		6:  uint8(13),
		7:  uint8(6),
		8:  uint8(1),
		9:  uint8(12),
		11: uint8(2),
		12: uint8(11),
		13: uint8(7),
		14: uint8(5),
		15: uint8(3),
	},
}

func Blake2b_compress_block(tls *libc.TLS, h uintptr, block uintptr, t0 uint64_t, t1 uint64_t, f0 uint64_t, f1 uint64_t) {
	var i, r int32
	var m, v [16]uint64_t
	_, _, _, _ = i, m, r, v
	i = 0
	for {
		if !(i < int32(16)) {
			break
		}
		m[i] = **(**uint64_t)(__ccgo_up(block + uintptr(i)*8))
		goto _1
	_1:
		;
		i = i + 1
	}
	i = 0
	for {
		if !(i < int32(8)) {
			break
		}
		v[i] = **(**uint64_t)(__ccgo_up(h + uintptr(i)*8))
		goto _2
	_2:
		;
		i = i + 1
	}
	v[int32(8)] = uint64(0x6a09e667f3bcc908)
	v[int32(9)] = uint64(0xbb67ae8584caa73b)
	v[int32(10)] = uint64(0x3c6ef372fe94f82b)
	v[int32(11)] = uint64(0xa54ff53a5f1d36f1)
	v[int32(12)] = uint64(0x510e527fea90715d) ^ t0
	v[int32(13)] = uint64(0x9b05688c2b3e6c1f) ^ t1
	v[int32(14)] = uint64(0x1f83d9abfb41bd6b) ^ f0
	v[int32(15)] = uint64(0x5be0cd19137e2179) ^ f1
	r = 0
	for {
		if !(r < int32(12)) {
			break
		}
		v[0] = v[0] + v[int32(4)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(0))))]
		v[int32(12)] = (v[int32(12)]^v[0])>>libc.Int32FromInt32(32) ^ (v[int32(12)]^v[0])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(8)] = v[int32(8)] + v[int32(12)]
		v[int32(4)] = (v[int32(4)]^v[int32(8)])>>libc.Int32FromInt32(24) ^ (v[int32(4)]^v[int32(8)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[0] = v[0] + v[int32(4)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(0)+libc.Int32FromInt32(1))))]
		v[int32(12)] = (v[int32(12)]^v[0])>>libc.Int32FromInt32(16) ^ (v[int32(12)]^v[0])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(8)] = v[int32(8)] + v[int32(12)]
		v[int32(4)] = (v[int32(4)]^v[int32(8)])>>libc.Int32FromInt32(63) ^ (v[int32(4)]^v[int32(8)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(1)] = v[int32(1)] + v[int32(5)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(1))))]
		v[int32(13)] = (v[int32(13)]^v[int32(1)])>>libc.Int32FromInt32(32) ^ (v[int32(13)]^v[int32(1)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(9)] = v[int32(9)] + v[int32(13)]
		v[int32(5)] = (v[int32(5)]^v[int32(9)])>>libc.Int32FromInt32(24) ^ (v[int32(5)]^v[int32(9)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(1)] = v[int32(1)] + v[int32(5)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(1)+libc.Int32FromInt32(1))))]
		v[int32(13)] = (v[int32(13)]^v[int32(1)])>>libc.Int32FromInt32(16) ^ (v[int32(13)]^v[int32(1)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(9)] = v[int32(9)] + v[int32(13)]
		v[int32(5)] = (v[int32(5)]^v[int32(9)])>>libc.Int32FromInt32(63) ^ (v[int32(5)]^v[int32(9)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(2)] = v[int32(2)] + v[int32(6)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(2))))]
		v[int32(14)] = (v[int32(14)]^v[int32(2)])>>libc.Int32FromInt32(32) ^ (v[int32(14)]^v[int32(2)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(10)] = v[int32(10)] + v[int32(14)]
		v[int32(6)] = (v[int32(6)]^v[int32(10)])>>libc.Int32FromInt32(24) ^ (v[int32(6)]^v[int32(10)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(2)] = v[int32(2)] + v[int32(6)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(2)+libc.Int32FromInt32(1))))]
		v[int32(14)] = (v[int32(14)]^v[int32(2)])>>libc.Int32FromInt32(16) ^ (v[int32(14)]^v[int32(2)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(10)] = v[int32(10)] + v[int32(14)]
		v[int32(6)] = (v[int32(6)]^v[int32(10)])>>libc.Int32FromInt32(63) ^ (v[int32(6)]^v[int32(10)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(3)] = v[int32(3)] + v[int32(7)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(3))))]
		v[int32(15)] = (v[int32(15)]^v[int32(3)])>>libc.Int32FromInt32(32) ^ (v[int32(15)]^v[int32(3)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(11)] = v[int32(11)] + v[int32(15)]
		v[int32(7)] = (v[int32(7)]^v[int32(11)])>>libc.Int32FromInt32(24) ^ (v[int32(7)]^v[int32(11)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(3)] = v[int32(3)] + v[int32(7)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(3)+libc.Int32FromInt32(1))))]
		v[int32(15)] = (v[int32(15)]^v[int32(3)])>>libc.Int32FromInt32(16) ^ (v[int32(15)]^v[int32(3)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(11)] = v[int32(11)] + v[int32(15)]
		v[int32(7)] = (v[int32(7)]^v[int32(11)])>>libc.Int32FromInt32(63) ^ (v[int32(7)]^v[int32(11)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[0] = v[0] + v[int32(5)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(4))))]
		v[int32(15)] = (v[int32(15)]^v[0])>>libc.Int32FromInt32(32) ^ (v[int32(15)]^v[0])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(10)] = v[int32(10)] + v[int32(15)]
		v[int32(5)] = (v[int32(5)]^v[int32(10)])>>libc.Int32FromInt32(24) ^ (v[int32(5)]^v[int32(10)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[0] = v[0] + v[int32(5)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(4)+libc.Int32FromInt32(1))))]
		v[int32(15)] = (v[int32(15)]^v[0])>>libc.Int32FromInt32(16) ^ (v[int32(15)]^v[0])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(10)] = v[int32(10)] + v[int32(15)]
		v[int32(5)] = (v[int32(5)]^v[int32(10)])>>libc.Int32FromInt32(63) ^ (v[int32(5)]^v[int32(10)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(1)] = v[int32(1)] + v[int32(6)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(5))))]
		v[int32(12)] = (v[int32(12)]^v[int32(1)])>>libc.Int32FromInt32(32) ^ (v[int32(12)]^v[int32(1)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(11)] = v[int32(11)] + v[int32(12)]
		v[int32(6)] = (v[int32(6)]^v[int32(11)])>>libc.Int32FromInt32(24) ^ (v[int32(6)]^v[int32(11)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(1)] = v[int32(1)] + v[int32(6)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(5)+libc.Int32FromInt32(1))))]
		v[int32(12)] = (v[int32(12)]^v[int32(1)])>>libc.Int32FromInt32(16) ^ (v[int32(12)]^v[int32(1)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(11)] = v[int32(11)] + v[int32(12)]
		v[int32(6)] = (v[int32(6)]^v[int32(11)])>>libc.Int32FromInt32(63) ^ (v[int32(6)]^v[int32(11)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(2)] = v[int32(2)] + v[int32(7)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(6))))]
		v[int32(13)] = (v[int32(13)]^v[int32(2)])>>libc.Int32FromInt32(32) ^ (v[int32(13)]^v[int32(2)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(8)] = v[int32(8)] + v[int32(13)]
		v[int32(7)] = (v[int32(7)]^v[int32(8)])>>libc.Int32FromInt32(24) ^ (v[int32(7)]^v[int32(8)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(2)] = v[int32(2)] + v[int32(7)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(6)+libc.Int32FromInt32(1))))]
		v[int32(13)] = (v[int32(13)]^v[int32(2)])>>libc.Int32FromInt32(16) ^ (v[int32(13)]^v[int32(2)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(8)] = v[int32(8)] + v[int32(13)]
		v[int32(7)] = (v[int32(7)]^v[int32(8)])>>libc.Int32FromInt32(63) ^ (v[int32(7)]^v[int32(8)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		v[int32(3)] = v[int32(3)] + v[int32(4)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(7))))]
		v[int32(14)] = (v[int32(14)]^v[int32(3)])>>libc.Int32FromInt32(32) ^ (v[int32(14)]^v[int32(3)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(32))
		v[int32(9)] = v[int32(9)] + v[int32(14)]
		v[int32(4)] = (v[int32(4)]^v[int32(9)])>>libc.Int32FromInt32(24) ^ (v[int32(4)]^v[int32(9)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(24))
		v[int32(3)] = v[int32(3)] + v[int32(4)] + m[**(**uint8_t)(__ccgo_up(uintptr(unsafe.Pointer(&blake2b_sigma)) + uintptr(r)*16 + uintptr(libc.Int32FromInt32(2)*libc.Int32FromInt32(7)+libc.Int32FromInt32(1))))]
		v[int32(14)] = (v[int32(14)]^v[int32(3)])>>libc.Int32FromInt32(16) ^ (v[int32(14)]^v[int32(3)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(16))
		v[int32(9)] = v[int32(9)] + v[int32(14)]
		v[int32(4)] = (v[int32(4)]^v[int32(9)])>>libc.Int32FromInt32(63) ^ (v[int32(4)]^v[int32(9)])<<(libc.Int32FromInt32(64)-libc.Int32FromInt32(63))
		goto _3
	_3:
		;
		r = r + 1
	}
	i = 0
	for {
		if !(i < int32(8)) {
			break
		}
		**(**uint64_t)(__ccgo_up(h + uintptr(i)*8)) = **(**uint64_t)(__ccgo_up(h + uintptr(i)*8)) ^ v[i] ^ v[i+int32(8)]
		goto _4
	_4:
		;
		i = i + 1
	}
}

func __ccgo_up(n uintptr) unsafe.Pointer {
	return unsafe.Pointer(&n)
}
