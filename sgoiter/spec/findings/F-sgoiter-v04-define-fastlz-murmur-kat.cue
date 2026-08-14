package findings

F_sgoiter_v04_define_fastlz_murmur_kat: #Finding & {
	id:     "F-sgoiter-v04-define-fastlz-murmur-kat"
	kernel: "flz_hash|murmur3_x86_32"
	stage:  "dogfood"
	symptom: "Macros HASH_* et void* bloquent harvest fastlz ; Murmur sans oracle C."
	evidence: {
		file_line: "front/preprocess.go foldDefines ; testdata/c/fastlz_lab.c ; murmur_kat_test.go"
		kat:       "pass"
	}
	lever:   "front"
	action:  "fold #define object+LIKELY ; void*→[]byte ; KAT Murmur vs gcc -O0 vecteurs."
	status:  "landed"
}
