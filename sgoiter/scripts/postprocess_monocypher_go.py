#!/usr/bin/env python3
"""[DEPRECATED / OBSOLÈTE] Post-process sgoiter monocypher emit.

La chaîne canonique d'émission sgoiter intègre désormais nativement toutes les passes
d'idiomatisation (-exclude unifié, archArrayNotSlice, archBuiltinMinMax, etc.).
Ce script n'est conservé que pour compatibilité d'outillage historique.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path


def postprocess(t: str) -> str:
    t = t.replace("var base_point = [1]byte{9}", "var base_point = [32]byte{9}")
    t = re.sub(
        r"var (zero|key_block|crypto_blake2b_keyed_init_key_block) = make\(\[\]byte, 128\)",
        r"var \1_arr [128]byte\nvar \1 = \1_arr[:]",
        t,
    )
    t = re.sub(
        r"var crypto_x25519_public_key_base_point = (?:\[\]byte\{9\}|func\(\) \[\]byte \{.*?\(\))",
        "var crypto_x25519_public_key_base_point_arr = [32]byte{9}\nvar crypto_x25519_public_key_base_point = crypto_x25519_public_key_base_point_arr[:]",
        t,
    )

    def pad_fe(m: re.Match[str]) -> str:
        name, body = m.group(1), m.group(2)
        nums = [x.strip() for x in body.split(",") if x.strip()]
        while len(nums) < 10:
            nums.append("0")
        return f"var {name} = []int{{{', '.join(nums[:10])}}}"

    t = re.sub(
        r"var (fe_one|sqrtm1|d|D2|lop_x|lop_y|ufactor|A2|A) = \[\]int\{([^}]*)\}",
        pad_fe,
        t,
    )
    t = re.sub(
        r"\nfunc \w+\(args \.\.\.any\)[^{]*\{[^}]*was not harvested[^}]*\}\n",
        "\n",
        t,
    )
    t = re.sub(
        r"\nfunc \w+\(args \.\.\.any\) int \{[^}]*was not harvested[^}]*\}\n",
        "\n",
        t,
    )
    t = re.sub(r"\ntype Slide_ctx struct \{.*?\n\}", "\n", t, count=1, flags=re.S)
    for fn in (
        "Slide_init",
        "Slide_step",
        "Remove_l",
        "Mod_l",
        "Invsqrt",
        "Lookup_add",  # emit types comb as *Ge_precomp; hand ge_scalarmult_base.go
        "Crypto_argon2",  # hand_argon2.go (volatile casts)
        "Crypto_elligator_key_pair",  # hand_elligator_key_pair.go
        "Crypto_chacha20_djb",  # hand_chacha_simd.go
        "Crypto_x25519_dirty_small",  # hand_x25519_dirty_small.go
        "Poly_blocks",  # hand_poly1305_simd.go
        "Ge_scalarmult_base",  # ge_scalarmult_base.go
        "Crypto_eddsa_check_equation",  # eddsa_check_equation.go
        "Crypto_aead_write",  # hand_aead_fused.go (boucle fusionnée 1-pass)
    ):
        t = re.sub(rf"\nfunc {fn}\b.*?(?=\nfunc |\Z)", "\n", t, count=1, flags=re.S)
    return t


def main() -> None:
    if len(sys.argv) != 2:
        print("usage: postprocess_monocypher_go.py <file.go>", file=sys.stderr)
        sys.exit(2)
    p = Path(sys.argv[1])
    p.write_text(postprocess(p.read_text()))
    print("postprocessed", p)


if __name__ == "__main__":
    main()
