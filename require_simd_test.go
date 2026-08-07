//go:build !goexperiment.simd

package c2simd_test

import "testing"

// TestRequireGOEXPERIMENTSimd empêche le faux vert CI : sans GOEXPERIMENT=simd,
// les tests du cœur SIMD ne sont pas compilés. Ce test force l'échec explicite.
func TestRequireGOEXPERIMENTSimd(t *testing.T) {
	t.Fatal("c2simd: core SIMD suite requires GOEXPERIMENT=simd (got build without goexperiment.simd)")
}
