// F-sgoiter-ptr-index-forcond — 2026-08-11
// Build-green ≠ bit-correct : lecture visuelle des 12 dogfood.
//
// Bugs clos :
// 1. Cast (uint64_t*)p héritait elemIndex du uint8* → index mot non scalé.
//    Fix : cast change de type ⇒ scale=sizeof, elemIndex=false sauf même type.
// 2. Store LE faisait idx*8 alors que load attendait idx déjà byte-scalé.
//    Fix : store []byte wide = idx byte (front scale les deux).
// 3. Cond for (i+8<=len) figée dans ForInit → boucle infinie.
//    Fix : ForCondPrep re-éval chaque itération ; offSlot init reste ForInit once.
// 4. ptr += N reslice sans bumper offSlot → siphash panic len>=8.
//    Fix : si offSlotSet, avancer offSlot uniquement.
//
// KAT runtime : fnv1a_64, crc32_ieee, fast_xor, siphash24 (kernel_kat_test.go).
// monocypher AEAD dogfood inchangé (PASS).
{
	id: "F-sgoiter-ptr-index-forcond"
	status: "closed"
	date: "2026-08-11"
}
