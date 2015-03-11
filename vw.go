package main

// #cgo CFLAGS: -I/usr/local/include/vowpalwabbit
// #cgo darwin LDFLAGS:  -L/usr/local/lib -lvw_c_wrapper -lvw -lallreduce -lboost_program_options -lz -lstdc++
// #cgo linux LDFLAGS: /usr/local/lib/libvw_c_wrapper.a /usr/local/lib/libvw.a /usr/local/lib/liballreduce.a /usr/lib/libboost_program_options.a -lz -lstdc++
/*
#include <stdlib.h>
#define bool int
#define true (1)
#define false (0)
#include "vwdll.h"
*/
import "C"
import (
	"math"
	"unsafe"
)

type VW struct {
	h       C.VW_HANDLE
	cmdline *C.char
}

func logistic01(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func NewVW(fn string) *VW {

	vw := &VW{}
	vw.cmdline = C.CString("--quiet -t -i " + fn)
	vw.h = C.VW_InitializeA(vw.cmdline)

	return vw
}

func (vw *VW) Finish() {
	C.VW_Finish(vw.h)
	C.free(unsafe.Pointer(vw.cmdline))
}

func (vw *VW) Predict(ex string) float64 {

	exampleC := C.CString(ex)
	defer C.free(unsafe.Pointer(exampleC))

	example := C.VW_ReadExampleA(vw.h, exampleC)
	pred := C.VW_Predict(vw.h, example)

	C.VW_FinishExample(vw.h, example)

	return logistic01(float64(pred))
}
