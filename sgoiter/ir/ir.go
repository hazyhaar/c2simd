// Package ir — représentation intermédiaire sgoiter (pré-Go).
package ir

// Op is an IR instruction opcode.
type Op uint8

const (
	OpNop Op = iota
	OpConst
	OpMov
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
	OpReturn
	OpCall
	OpLoad   // dst = *ptr  or ptr[idx]  Args:[ptr] or [ptr,idx]
	OpStore  // *ptr = val or ptr[idx]=val  Args:[ptr,val] or [ptr,idx,val]
	OpAlloca // dst = local array; Imm=len, Elem=elem type
	OpNot    // dst = ^x (bitwise not)
	OpField  // dst = arg0->Sym  (struct field; if array field, dst is slice header)
	OpFStore // arg0->Sym = arg1  or arg0->Sym[arg2]=arg1
	OpBreak
	OpContinue
	OpGoto
	OpLabel
	// Opcodes vectoriels SIMD (Go 1.27 archsimd / portable simd)
	OpVecLoad
	OpVecStore
	OpVecAdd
	OpVecSub
	OpVecMul
	OpVecFMA
	OpVecCmpEq
	OpVecToBits
	OpVecReduceSum
)

func (o Op) String() string {
	names := []string{
		"Nop", "Const", "Mov", "Add", "Sub", "Mul", "Div", "Mod",
		"And", "Or", "Xor", "Shl", "Shr", "Return", "Call",
		"Load", "Store", "Alloca", "Not", "Field", "FStore",
		"Break", "Continue", "Goto", "Label",
		"VecLoad", "VecStore", "VecAdd", "VecSub", "VecMul", "VecFMA",
		"VecCmpEq", "VecToBits", "VecReduceSum",
	}
	if int(o) < len(names) {
		return names[o]
	}
	return "?"
}

// TypeName is a subset scalar / pointer-elem / vector type name.
type TypeName string

const (
	TypInt        TypeName = "int"
	TypInt8       TypeName = "int8"
	TypInt16      TypeName = "int16"
	TypInt32      TypeName = "int32"
	TypInt64      TypeName = "int64"
	TypUint8      TypeName = "uint8"
	TypUint16     TypeName = "uint16"
	TypUint32     TypeName = "uint32"
	TypUint64     TypeName = "uint64"
	TypFloat32    TypeName = "float32"
	TypFloat64    TypeName = "float64"
	TypBool       TypeName = "bool"
	TypString     TypeName = "string"
	TypUint128    TypeName = "uint128"
	TypVoid       TypeName = "void"
	// Types vectoriels SIMD
	TypVecUint8x16   TypeName = "vec_uint8x16"
	TypVecUint8x32   TypeName = "vec_uint8x32"
	TypVecUint32x4   TypeName = "vec_uint32x4"
	TypVecUint32x8   TypeName = "vec_uint32x8"
	TypVecUint64x2   TypeName = "vec_uint64x2"
	TypVecUint64x4   TypeName = "vec_uint64x4"
	TypVecFloat32x4  TypeName = "vec_float32x4"
	TypVecFloat32x8  TypeName = "vec_float32x8"
	TypVecFloat64x2  TypeName = "vec_float64x2"
	TypVecFloat64x4  TypeName = "vec_float64x4"
)

// Value is a virtual register index, or -1 for none.
type Value int

const NoVal Value = -1

// Instr is one flat instruction.
type Instr struct {
	Op   Op      `json:"op"`
	Dst  Value   `json:"dst,omitempty"`
	Args []Value `json:"args,omitempty"`
	Imm  int64   `json:"imm,omitempty"`
	Sym  string  `json:"sym,omitempty"`
	// Elem type for Load/Store (byte width hint).
	Elem TypeName `json:"elem,omitempty"`
}

// StmtKind tags structured statements.
type StmtKind uint8

const (
	SKInstr StmtKind = iota
	SKFor
	SKSwitch
	SKDoWhile
	SKIf
)

// Stmt is an instruction or structured control-flow.
type Stmt struct {
	Kind StmtKind `json:"kind"`

	// SKInstr
	Ins Instr `json:"ins,omitempty"`

	// SKFor — Init once; CondPrep re-run each iteration before Cond; Post after body.
	ForInit []Instr `json:"for_init,omitempty"`
	// ForCondPrep: instrs that materialize CondLeft/Right (e.g. i+8) — must re-eval each iter.
	ForCondPrep []Instr `json:"for_cond_prep,omitempty"`
	// CondLeft OP CondRight — if CondOp empty, CondLeft is truthiness
	CondLeft  Value   `json:"cond_left,omitempty"`
	CondOp    string  `json:"cond_op,omitempty"` // "!=", "<", ">", "==", ""
	CondRight Value   `json:"cond_right,omitempty"`
	ForPost   []Instr `json:"for_post,omitempty"`
	ForBody   []Stmt  `json:"for_body,omitempty"`

	// SKSwitch
	SwitchPrep  []Instr      `json:"switch_prep,omitempty"`
	SwitchOn    Value        `json:"switch_on,omitempty"`
	SwitchCases []SwitchCase `json:"switch_cases,omitempty"`

	// SKDoWhile — body then condition (C do { } while (cond))
	DoBody []Stmt `json:"do_body,omitempty"`

	// SKIf — Cond* + ThenBody [ElseBody]; Cond prep in ForInit
	ThenBody []Stmt `json:"then_body,omitempty"`
	ElseBody []Stmt `json:"else_body,omitempty"`
	CondNot  bool   `json:"cond_not,omitempty"` // if (!x)
}

// SwitchCase is one case label. Fall is C implicit fallthrough (no break/return).
type SwitchCase struct {
	// Labels empty ⇒ default
	Labels []int64 `json:"labels,omitempty"`
	Body   []Stmt  `json:"body,omitempty"`
	Fall   bool    `json:"fall,omitempty"`
}

// Func is one function.
type Func struct {
	Name   string   `json:"name"`
	Result TypeName `json:"result"`
	Params []Param  `json:"params,omitempty"`
	NVals  int      `json:"nvals"`
	// Stmts is the structured body (v0.3+).
	Stmts []Stmt `json:"stmts,omitempty"`
	// Body is flat instrs (v0.1–0.2 / rules); filled by Flatten or legacy.
	Body []Instr `json:"body,omitempty"`
	// Static marks a C `static` function: an internal helper, not an entry
	// point. Root-closure emission keeps it only when something reachable calls it.
	Static bool `json:"static,omitempty"`
	// LocalNames maps virtual register values to original C variable names.
	LocalNames map[Value]string `json:"local_names,omitempty"`
}

// Param is a function parameter.
type Param struct {
	Name       string   `json:"name"`
	Type       TypeName `json:"type"` // element type or struct name
	Ptr        bool     `json:"ptr,omitempty"`
	ArrayLen   int      `json:"array_len,omitempty"` // T name[N] → decays to ptr, N>0
	StructName string   `json:"struct_name,omitempty"`
	Reg        Value    `json:"reg"`
}

// Global is a module-level constant (string/byte table).
type Global struct {
	Name string `json:"name"`
	// Data is the string/byte contents (Go string literal body, unescaped simple).
	Data    string   `json:"data,omitempty"`
	Type    TypeName `json:"type,omitempty"` // element; default uint8
	ZeroLen int      `json:"zero_len,omitempty"`
	InitCSV string   `json:"init_csv,omitempty"` // raw C init list (flattened)
	Value   int64    `json:"value,omitempty"`
	// Cols > 0 : table was T name[Rows][Cols], stored row-major flat; index [i][j] → i*Cols+j.
	Cols int `json:"cols,omitempty"`
	Rows int `json:"rows,omitempty"`
}

// StructField is one field in a harvested C struct.
type StructField struct {
	Name     string   `json:"name"`
	Type     TypeName `json:"type"`
	ArrayLen int      `json:"array_len,omitempty"`
}

// StructType is a typedef struct layout.
type StructType struct {
	Name   string        `json:"name"`
	Fields []StructField `json:"fields"`
}

// Module is a compilation unit.
type Module struct {
	Name    string       `json:"name"`
	Globals []Global     `json:"globals,omitempty"`
	Structs []StructType `json:"structs,omitempty"`
	Funcs   []Func       `json:"funcs"`
}

// NewModule returns an empty named module.
func NewModule(name string) *Module {
	return &Module{Name: name}
}

// Alloc grows the vreg space and returns the new register.
func (f *Func) Alloc() Value {
	v := Value(f.NVals)
	f.NVals++
	return v
}

// Flatten appends all instructions from Stmts into Body (for rules that scan flat).
func (f *Func) Flatten() {
	f.Body = nil
	var walk func([]Stmt)
	walk = func(ss []Stmt) {
		for _, s := range ss {
			switch s.Kind {
			case SKInstr:
				f.Body = append(f.Body, s.Ins)
			case SKFor:
				f.Body = append(f.Body, s.ForInit...)
				f.Body = append(f.Body, s.ForCondPrep...)
				walk(s.ForBody)
				f.Body = append(f.Body, s.ForPost...)
			case SKSwitch:
				f.Body = append(f.Body, s.SwitchPrep...)
				for _, c := range s.SwitchCases {
					walk(c.Body)
				}
			case SKDoWhile:
				walk(s.DoBody)
			case SKIf:
				f.Body = append(f.Body, s.ForInit...)
				walk(s.ThenBody)
				walk(s.ElseBody)
			}
		}
	}
	walk(f.Stmts)
}

// EnsureStmts migrates legacy Body-only funcs to Stmts.
func (f *Func) EnsureStmts() {
	if len(f.Stmts) > 0 {
		return
	}
	for _, ins := range f.Body {
		f.Stmts = append(f.Stmts, Stmt{Kind: SKInstr, Ins: ins})
	}
}
