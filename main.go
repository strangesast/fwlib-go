package main

/*
#cgo CFLAGS: -I./src
#cgo LDFLAGS: -L./extern/fwlib -lfwlib32 -Wl,-rpath=./extern/fwlib
#include "extern/fwlib/fwlib32.h"
#include <stdlib.h>
*/
import "C"

import (
	"flag"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/config"
	"log"
	"os"
	"unsafe"
)

type cfg struct {
	IP   string
	Port int
}

type kafka_cfg struct {
	BootstrapServers string
	InputTopic       string
}

func main() {
	var config_fname string

	flag.StringVar(&config_fname, "config", "config.yaml", "")
	flag.Parse()

	config_file, err := os.Open(config_fname)
	if err != nil {
		panic(err)
	}
	provider, err := config.NewYAML(config.Source(config_file))
	if err != nil {
		panic(err)
	}

	focas_debug_fname_val, err := provider.Get("focas_debug_fname").WithDefault("focas.log")
	if err != nil {
		panic(err)
	}
	var focas_debug_fname string
	err = focas_debug_fname_val.Populate(&focas_debug_fname)
	if err != nil {
		panic(err)
	}

	focas_debug_loglevel_val, err := provider.Get("focas_debug_loglevel").WithDefault(0)
	if err != nil {
		panic(err)
	}
	var focas_debug_loglevel int
	err = focas_debug_loglevel_val.Populate(&focas_debug_loglevel)
	if err != nil {
		panic(err)
	}

	log_fname := C.CString(focas_debug_fname)
	defer C.free(unsafe.Pointer(log_fname))
	if ret := C.cnc_startupprocess(C.long(focas_debug_loglevel), log_fname); ret != C.EW_OK {
		log.Fatalln("cnc_startupprocess failed")
	}

	var config cfg
	err = provider.Get("machine").Populate(&config)
	if err != nil {
		panic(err)
	}
	ip := C.CString(config.IP)
	defer C.free(unsafe.Pointer(ip))
	port := (C.ushort)(config.Port)

	var libh C.ushort
	if ret := C.cnc_allclibhndl3(ip, port, 10, &libh); ret != C.EW_OK {
		log.Fatalf("cnc_allclibhndl3 failed (%d)\n", ret)
	}

	var cnc_ids [4]uint32
	// on x64 linux, ulongs are typically 64 bits not the 32 that fwlib expects
	if ret := C.cnc_rdcncid(libh, (*C.ulong)(unsafe.Pointer(&cnc_ids[0]))); ret != C.EW_OK {
		log.Fatalf("cnc_rdcncid failed (%d)\n", ret)
	}
	fmt.Printf("\nmachine id: %08x-%08x-%08x-%08x\n", cnc_ids[0], cnc_ids[1], cnc_ids[2], cnc_ids[3])

	var sysinfo C.ODBSYS
	if ret := C.cnc_sysinfo(libh, &sysinfo); ret != C.EW_OK {
		log.Fatalf("cnc_sysinfo failed (%d)\n", ret)
	}

	// the following are not null terminated so GoStringN is required
	fmt.Printf("\naddinfo:   %d\n", sysinfo.addinfo)
	fmt.Printf("max_axis:  %d\n", sysinfo.max_axis)
	fmt.Printf("cnc_type:  %s\n", C.GoStringN(&sysinfo.cnc_type[0], 2))
	fmt.Printf("mt_type:   %s\n", C.GoStringN(&sysinfo.mt_type[0], 2))
	fmt.Printf("series:    %s\n", C.GoStringN(&sysinfo.series[0], 4))
	fmt.Printf("version:   %s\n", C.GoStringN(&sysinfo.version[0], 4))
	fmt.Printf("axes:      %s\n", C.GoStringN(&sysinfo.axes[0], 2))

	var axes [C.MAX_AXIS]C.ODBAXISNAME
	var c C.short = C.MAX_AXIS
	if ret := C.cnc_rdaxisname(libh, &c, (*C.ODBAXISNAME)(unsafe.Pointer(&axes))); ret != C.EW_OK {
		log.Fatalf("cnc_rdaxisname failed (%d)\n", ret)
	}
	axis_count := int(c)
	axes_names := make([]string, axis_count)
	fmt.Printf("\naxis_count: %d\n", axis_count)
	for i := 0; i < axis_count; i++ {
		s := C.GoStringN(&axes[i].name, 1)
		axes_names[i] = s
	}
	fmt.Printf("axes names: %v\n", axes_names)

	if ret := C.cnc_freelibhndl(libh); ret != C.EW_OK {
		log.Fatalf("cnc_freelibhndl failed (%d)\n", ret)
	}

	if ret := C.cnc_exitprocess(); ret != C.EW_OK {
		log.Fatalf("cnc_exitprocess failed (%d)\n", ret)
	}
}
