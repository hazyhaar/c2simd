package emit_test

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
)

func TestLocalAggregateInitializer(t *testing.T) {
	cSrc := `
#include <stdint.h>

uint32_t test_local_init(uint32_t x) {
    uint32_t table[4] = {0x12345678, 0x9abcdef0, 42, 100};
    return table[x & 3];
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err != nil {
		t.Fatalf("front parse failed: %v", err)
	}
	if res.Module == nil || len(res.Module.Funcs) == 0 {
		t.Fatalf("no funcs harvested")
	}

	goCode, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if !strings.Contains(goCode, "0x12345678") || !strings.Contains(goCode, "0x9abcdef0") {
		t.Fatalf("local aggregate initializer values missing in emitted code:\n%s", goCode)
	}
	if strings.Contains(goCode, "var table [4]uint32\n") || strings.Contains(goCode, "var _arr_table [4]uint32\n") {
		t.Fatalf("zero-initialized table emitted instead of initialized aggregate:\n%s", goCode)
	}
}

func TestOpMulWidening(t *testing.T) {
	cSrc := `
#include <stdint.h>
#include <stddef.h>

uint64_t test_mul_widen(const uint32_t *L, uint8_t x) {
    uint64_t carry = 0;
    for (size_t i = 0; i < 8; i++) {
        carry += (uint64_t)L[i] * (x & 7);
    }
    return carry;
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err != nil {
		t.Fatalf("front parse failed: %v", err)
	}

	goCode, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	// The operands must be cast before multiplication
	if !strings.Contains(goCode, "uint64(L[") {
		t.Fatalf("64-bit operand widening missing in emitted code:\n%s", goCode)
	}
}

func TestLocalAggregateInitializer2DNested(t *testing.T) {
	cSrc := `
#include <stdint.h>

uint32_t test_2d_nested(uint32_t i, uint32_t j) {
    uint32_t matrix[2][2] = {
        {0x10, 0x20},
        {0x30, 0x40}
    };
    return matrix[i & 1][j & 1];
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err != nil {
		t.Fatalf("front parse failed: %v", err)
	}
	if res.Module == nil || len(res.Module.Funcs) == 0 {
		t.Fatalf("no funcs harvested")
	}

	goCode, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if !strings.Contains(goCode, "0x10") || !strings.Contains(goCode, "0x40") {
		t.Fatalf("nested 2D initializer values missing in emitted code:\n%s", goCode)
	}
}

func TestCopyClearLoopTransformation(t *testing.T) {
	cSrc := `
#include <stdint.h>
#include <stddef.h>

void test_copy_clear(uint8_t *dst, const uint8_t *src, uint8_t *wipe_target) {
    for (size_t i = 0; i < 32; i++) {
        dst[i] = src[i];
    }
    for (size_t i = 0; i < 16; i++) {
        wipe_target[i] = 0;
    }
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err != nil {
		t.Fatalf("front parse failed: %v", err)
	}

	goCode, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if !strings.Contains(goCode, "copy(dst[:32], src[:32])") {
		t.Fatalf("copy() transformation missing in emitted code:\n%s", goCode)
	}
	if !strings.Contains(goCode, "clear(wipe_target[:16])") {
		t.Fatalf("clear() transformation missing in emitted code:\n%s", goCode)
	}
}

func TestLocalStructInitializer(t *testing.T) {
	cSrc := `
#include <stdint.h>

struct Point {
    uint32_t x;
    uint32_t y;
};

uint32_t test_struct_init(uint32_t offset) {
    struct Point pt = {100, 200};
    return pt.x + pt.y + offset;
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err != nil {
		t.Fatalf("front parse failed: %v", err)
	}
	if res.Module == nil || len(res.Module.Funcs) == 0 {
		t.Fatalf("no funcs harvested")
	}

	goCode, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if !strings.Contains(goCode, "Point{100, 200}") {
		t.Fatalf("struct initializer missing in emitted code:\n%s", goCode)
	}
}

func TestUnsupportedAggregateInitializerFailsLoudly(t *testing.T) {
	cSrc := `
#include <stdint.h>

uint32_t test_unsupported(uint32_t x) {
    mystery_unrecognized_t mystery = {1, 2, 3};
    return x;
}
`
	res, err := front.ParsePartial(cSrc, "test_pkg")
	if err == nil && res.Module != nil && len(res.Module.Funcs) > 0 {
		t.Fatalf("expected failure or skipped func on unrecognized aggregate initializer")
	}
}
