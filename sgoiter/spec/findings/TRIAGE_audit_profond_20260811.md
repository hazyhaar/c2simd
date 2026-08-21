# Triage audit profond — statut 2026-08-11 (post-8a46a88 / 3be6e74 / TypInt64)

| Code | Statut | Preuve |
|------|--------|--------|
| C1 wipe-before-return | **landed** | `3be6e74`, `TestFeIsOdd_WipeCheck` |
| C2 shift32-trunc | **landed** | `3e292af`, `TestChacha20IETF_VsC` |
| C3 precedence mask | **landed** | `8a46a88` needsParen, yeux `s[29:]` |
| C4 ptr advance *p/p++ | **landed** | `8a46a88`, `TestPtrCursor`, `PolyRemainderThenData` |
| C5 alias cache | résolu / non re-prouvé | lié offSlot |
| C6 tweetnacl param | hors scope banc | tribench tweetnacl 11/11 surface verify |
| C7 i64 unsigned | **landed** | `TypInt64` + mapType ; Fe temps int64 |
| C8–C9 libinjection | **smoke-only** | `F-sgoiter-libinjection-smoke-only` ; pas bit-exact sgoiter |
| C10–C12 | ouverts / secondaires | oracles à étendre si besoin |
| C13–C16 forme | validés | T1–T16 |
| C17 md5 fixture | fixture C | fidélité OK |
| C18 LE doctrine | décision | binary.LittleEndian |
