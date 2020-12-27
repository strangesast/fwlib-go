package main

/*
#cgo CFLAGS: -I./src
#cgo LDFLAGS: -L./extern/fwlib -lfwlib32 -Wl,-rpath=./extern/fwlib
#include "extern/fwlib/fwlib32.h"
#include <stdlib.h>
#include <stdint.h>

typedef struct iodbpsd_t {
    int16_t   datano;
    int16_t   type;
    union {
        char    cdata ;
        int16_t idata ;
        int32_t ldata ;
        char    cdatas[MAX_AXIS] ;
        int16_t idatas[MAX_AXIS] ;
        int32_t ldatas[MAX_AXIS] ;
    } u ;
} iodbpsd_t ;

typedef struct odbdy2_t {
    short   dummy ;
    short   axis ;
    long    alarm ;
    long    prgnum ;
    long    prgmnum ;
    long    seqnum ;
    long    actf ;
    long    acts ;
    union {
        struct {
            long    absolute[MAX_AXIS] ;
            long    machine[MAX_AXIS] ;
            long    relative[MAX_AXIS] ;
            long    distance[MAX_AXIS] ;
        } faxis ;
        struct {
            long    absolute ;
            long    machine ;
            long    relative ;
            long    distance ;
        } oaxis ;
    } pos ;
} ODBDY2_T ;
*/
import "C"

import (
	"fmt"
	// 	"go.uber.org/config"
	// 	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"errors"
	"log"
	"unsafe"
)

type Update struct {
	Key   string
	Value interface{}
}

type sig = func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error

func read_id(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var cnc_ids [4]uint32
		if ret := C.cnc_rdcncid(libh, (*C.ulong)(unsafe.Pointer(&cnc_ids[0]))); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_rdcncid failed (%d)", ret))
		}
		value := fmt.Sprintf("%08x-%08x-%08x-%08x", cnc_ids[0], cnc_ids[1], cnc_ids[2], cnc_ids[3])
		(*b)["id"] = value

		*c = append(*c, Update{"id", value})

		return nil
	}
}

func read_parameter(libh C.ushort, key string, num int) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var param C.iodbpsd_t

		if ret := C.cnc_rdparam(libh, (C.short)(num), C.ALL_AXES, 8, (*C.IODBPSD)(unsafe.Pointer(&param))); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_rdparam failed (%d)", ret))
		}
		last := (*a)[key]
		switch last.(type) {
		case C.long:
			value := *(*C.long)(unsafe.Pointer(&param.u))
			if last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		default:
			value := param.u
			(*b)[key] = value
			*c = append(*c, Update{key, value})
		}
		return nil
	}
}

type machine_info struct {
	addinfo  int
	max_axis int
	cnc_type string
	mt_type  string
	series   string
	version  string
	axes     string
}

func read_machine_info(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		// on x64 linux, ulongs are typically 64 bits not the 32 that fwlib expects
		const key = "info"
		var sysinfo C.ODBSYS
		if ret := C.cnc_sysinfo(libh, &sysinfo); ret != C.EW_OK {
			log.Fatalf("cnc_sysinfo failed (%d)\n", ret)
		}
		// the following are not null terminated so GoStringN is required
		value := machine_info{
			int(sysinfo.addinfo),
			int(sysinfo.max_axis),
			C.GoStringN(&sysinfo.cnc_type[0], 2),
			C.GoStringN(&sysinfo.mt_type[0], 2),
			C.GoStringN(&sysinfo.series[0], 4),
			C.GoStringN(&sysinfo.version[0], 4),
			C.GoStringN(&sysinfo.axes[0], 2),
		}
		(*b)[key] = value
		*c = append(*c, Update{key, value})

		return nil
	}

}

func read_axis_names(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		const key = "axis_names"
		var axes [C.MAX_AXIS]C.ODBAXISNAME
		var cnt C.short = C.MAX_AXIS
		if ret := C.cnc_rdaxisname(libh, &cnt, (*C.ODBAXISNAME)(unsafe.Pointer(&axes))); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_rdaxisname failed (%d)", ret))
		}
		axis_count := int(cnt)
		value := make([]string, axis_count)
		for i := 0; i < axis_count; i++ {
			s := C.GoString(&axes[i].name)
			value[i] = s
		}
		(*b)[key] = value
		*c = append(*c, Update{key, value})

		return nil
	}
}

func read_program_contents(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var program_contents []byte
		var program_size int = 0
		path := ""
		var _type C.short = 0

		_path := unsafe.Pointer(C.CString(path))
		defer C.free(_path)
		if ret := C.cnc_upstart4(libh, _type, (*C.char)(_path)); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_upstart4 failed (%d)", ret))
		}

		for {
			var l C.long = 1280
			buf := make([]byte, l)
			if ret := C.cnc_upload4(libh, &l, (*C.char)(unsafe.Pointer(&buf[0]))); ret == C.EW_BUFFER {
				continue
			} else if ret != C.EW_OK {
				break
			}
			if l > 0 {
				program_size += int(l)
				program_contents = append(program_contents, buf[0:l]...)
				if buf[l-1] == '%' {
					break
				}
			}
		}

		if ret := C.cnc_upend4(libh); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_upend4 failed (%d)", ret))
		}

		(*b)["program_contents"] = string(program_contents)
		(*b)["program_size"] = program_size

		return nil
	}
}

func read_status(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var odbst C.ODBST
		const key = "raw_status"
		if ret := C.cnc_statinfo(libh, &odbst); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_statinfo failed (%d)", ret))
		}
		value := raw_status{
			int(odbst.alarm),
			int(odbst.aut),
			int(odbst.edit),
			int(odbst.emergency),
			int(odbst.hdck),
			int(odbst.motion),
			int(odbst.mstb),
			int(odbst.run),
			int(odbst.tmmode),
		}

		(*b)[key] = value
		*c = append(*c, Update{key, value})

		return nil
	}
}

type raw_status struct {
	alarm     int
	aut       int
	edit      int
	emergency int
	hdck      int
	motion    int
	mstb      int
	run       int
	tmmode    int
}

func get_execution(s raw_status) string {
	if s.run == 3 || s.run == 4 {
		return "active"
	} else if s.run == 2 || s.motion == 2 || s.mstb != 0 {
		return "interrupted"
	} else if s.run == 0 {
		return "stopped"
	} else {
		return "ready"
	}
}

func read_exection(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var last string
		const key = "execution"
		if val, ok := (*a)["raw_status"]; ok {
			last = get_execution(val.(raw_status))
		}
		value := get_execution((*b)["raw_status"].(raw_status))
		if last != value {
			(*b)[key] = value
			*c = append(*c, Update{key, value})
		}

		return nil
	}
}

func get_mode(s raw_status) string {
	if s.aut == 5 || s.aut == 6 {
		return "manual"
	} else if s.aut == 0 || s.aut == 3 {
		return "manual_data_input"
	} else {
		return "automatic"
	}
}

func get_emergency(s raw_status) string {
	if s.emergency == 1 {
		return "triggered"
	} else {
		return "armed"
	}
}

func read_dynamic(libh C.ushort) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var dyn C.ODBDY2_T

		if ret := C.cnc_rddynamic2(libh, C.ALL_AXES, C.sizeof_ODBDY2_T, (*C.ODBDY2)(unsafe.Pointer(&dyn))); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_rddynamic2 failed (%d)", ret))
		}

		return nil
	}
}

func main() {
	var libh C.ushort

	log_fname := C.CString("focas.log")
	defer C.free(unsafe.Pointer(log_fname))
	if ret := C.cnc_startupprocess(0, log_fname); ret != C.EW_OK {
		log.Fatalf("cnc_startupprocess failed (%d)\n", ret)
	}

	ip := C.CString("localhost")
	defer C.free(unsafe.Pointer(ip))
	port := (C.ushort)(8193)

	if ret := C.cnc_allclibhndl3(ip, port, 10, &libh); ret != C.EW_OK {
		log.Fatalf("cnc_allclibhndl3 failed (%d)\n", ret)
	}

	// read config
	// loop on interval

	const PART_COUNT_PARAMETER = 6711

	fns := []sig{
		read_id(libh),
		read_parameter(libh, "part_count", PART_COUNT_PARAMETER),
		read_machine_info(libh),
		read_axis_names(libh),
		read_status(libh),
	}

	a := make(map[string]interface{})
	a["part_count"] = (C.long)(0)
	b := make(map[string]interface{})
	var c []Update

	for _, each := range fns {
		each(&a, &b, &c)
	}
	a = b
	b = make(map[string]interface{})
	c = c[:0]
	for _, each := range fns {
		each(&a, &b, &c)
	}

	fmt.Printf("a: %+v\n", a)
	fmt.Printf("b: %+v\n", b)
	fmt.Printf("c: %+v\n", c)

	if ret := C.cnc_freelibhndl(libh); ret != C.EW_OK {
		log.Fatalf("cnc_freelibhndl failed (%d)\n", ret)
	}

	if ret := C.cnc_exitprocess(); ret != C.EW_OK {
		log.Fatalf("cnc_exitprocess failed (%d)\n", ret)
	}
}
