package findings

F_sgoiter_ptr_add_callarg: #Finding & {
	id:     "F-sgoiter-ptr-add-callarg"
	kernel: "load32_le(src + i*4)"
	stage:  "front"
	symptom: "ptr+off en Mov ptr_alias sans offset → Load32_le(src) ignore i*4; v8 unused."
	evidence: {
		file_line: "front ptr+ → OpAdd Sym=ptr_add; emit reslice p[off:]"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Locals: alias+off meta (murmur blocks[-i]). Call args: matérialiser ptr_add reslice avant Call."
	status: "landed"
	notes:  "ptr_add systématique cassait murmur (reslice négatif). Séparer local vs call-arg."
}
