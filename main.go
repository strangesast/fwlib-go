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
	"log"
	"unsafe"
)

func main() {
	var libh C.ushort
	var cnc_ids [4]uint32
	var ret C.short

	log_fname := C.CString("focas.log")
	defer C.free(unsafe.Pointer(log_fname))
	ret = C.cnc_startupprocess(0, log_fname)
	if ret != C.EW_OK {
		log.Fatalln("cnc_startupprocess failed")
	}

	ip := C.CString("localhost")
	defer C.free(unsafe.Pointer(ip))
	port := (C.ushort)(8193)

	ret = C.cnc_allclibhndl3(ip, port, 10, &libh)
	if ret != C.EW_OK {
		log.Fatalf("cnc_allclibhndl3 failed (%d)\n", ret)
	}

	// on x64 linux, ulongs are typically 64 bits not the 32 that fwlib expects
	ret = C.cnc_rdcncid(libh, (*C.ulong)(unsafe.Pointer(&cnc_ids[0])))
	if ret != C.EW_OK {
		log.Fatalf("cnc_rdcncid failed (%d)\n", ret)
	}
	fmt.Printf("\nmachine id: %08x-%08x-%08x-%08x\n", cnc_ids[0], cnc_ids[1], cnc_ids[2], cnc_ids[3])

	var sysinfo C.ODBSYS
	ret = C.cnc_sysinfo(libh, &sysinfo)
	if ret != C.EW_OK {
		log.Fatalf("cnc_sysinfo failed (%d)\n", ret)
	}
	if ret == C.EW_OK {
		// the following are not null terminated so GoStringN is required
		fmt.Printf("\naddinfo:   %d\n", sysinfo.addinfo)
		fmt.Printf("max_axis:  %d\n", sysinfo.max_axis)
		fmt.Printf("cnc_type:  %s\n", C.GoStringN(&sysinfo.cnc_type[0], 2))
		fmt.Printf("mt_type:   %s\n", C.GoStringN(&sysinfo.mt_type[0], 2))
		fmt.Printf("series:    %s\n", C.GoStringN(&sysinfo.series[0], 4))
		fmt.Printf("version:   %s\n", C.GoStringN(&sysinfo.version[0], 4))
		fmt.Printf("axes:      %s\n", C.GoStringN(&sysinfo.axes[0], 2))
	}

	var axes [C.MAX_AXIS]C.ODBAXISNAME
	var c C.short = C.MAX_AXIS
	ret = C.cnc_rdaxisname(libh, &c, (*C.ODBAXISNAME)(unsafe.Pointer(&axes)))
	if ret != C.EW_OK {
		log.Fatalf("cnc_rdaxisname failed (%d)\n", ret)
	}
	axis_count := int(c)
	fmt.Printf("\naxis_count: %d\n", axis_count)
	for i := 0; i < axis_count; i++ {
		s := C.GoString(&axes[i].name)
		fmt.Printf("axis name: %s\n", s)
	}

	ret = C.cnc_freelibhndl(libh)
	if ret != C.EW_OK {
		log.Fatalln("cnc_freelibhndl failed")
	}

	ret = C.cnc_exitprocess()
	if ret != C.EW_OK {
		log.Fatalln("cnc_exitprocess failed (%d)", ret)
	}
}
