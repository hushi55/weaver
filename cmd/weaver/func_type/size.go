package func_type

type Var interface {
	Align() int64
	Size() int64
}

func Offsetsof(fields []Var) []int64 {
	offsets := make([]int64, len(fields))
	var o int64
	for i, f := range fields {
		a := f.Align()
		o = align(o, a)
		offsets[i] = o
		o += f.Size()
	}
	return offsets
}

// align returns the smallest y >= x such that y % a == 0.
func align(x, a int64) int64 {
	y := x + a - 1
	return y - y%a
}
