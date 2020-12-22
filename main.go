package main

/*
#cgo CFLAGS: -I./src
#cgo LDFLAGS: -L./extern/fwlib -lfwlib32 -Wl,-rpath=./extern/fwlib
#include "extern/fwlib/fwlib32.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func main() {
	var libh C.ushort
	var cnc_ids [4]uint32

	log_fname := C.CString("focas.log")
	defer C.free(unsafe.Pointer(log_fname))
	C.cnc_startupprocess(0, log_fname)

	ip := C.CString("localhost")
	defer C.free(unsafe.Pointer(ip))
	port := (C.ushort)(8193)

	var ret C.short
	ret = C.cnc_allclibhndl3(ip, port, 10, &libh)
	if ret == 0 {
		// on x64 linux, ulongs are typically 64 bits not the 32 that fwlib expects
		ret = C.cnc_rdcncid(libh, (*C.ulong)(unsafe.Pointer(&cnc_ids[0])))
		if ret == 0 {
			fmt.Printf("machine id: %08x-%08x-%08x-%08x\n", cnc_ids[0], cnc_ids[1], cnc_ids[2], cnc_ids[3])
		}
	}
	C.cnc_freelibhndl(libh)
	C.cnc_exitprocess()
}
