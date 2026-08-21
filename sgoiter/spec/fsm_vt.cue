package spec

#State:  "GROUND" | "ESCAPE" | "CSI_ENTRY" | "CSI_PARAM" | "OSC_STRING"
#Action: "EXECUTE" | "PRINT" | "PARAM" | "COLLECT" | "CLEAR" | "DISPATCH" | "IGNORE"

#Transition: {
	from:       #State
	byte_start: int
	byte_end:   int
	action:     #Action
	to:         *from | #State
}

fsm_transitions: [...#Transition] & [
	{from: "GROUND", byte_start: 0x00, byte_end: 0x17, action: "EXECUTE", to: "GROUND"},
	{from: "GROUND", byte_start: 0x20, byte_end: 0x7F, action: "PRINT", to: "GROUND"},
	{from: "GROUND", byte_start: 0x1B, byte_end: 0x1B, action: "CLEAR", to: "ESCAPE"},
]
