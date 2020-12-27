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
    int16_t dummy ;
    int16_t axis ;
    int32_t alarm ;
    int32_t prgnum ;
    int32_t prgmnum ;
    int32_t seqnum ;
    int32_t actf ;
    int32_t acts ;
    union {
        struct {
            int32_t absolute[MAX_AXIS] ;
            int32_t machine[MAX_AXIS] ;
            int32_t relative[MAX_AXIS] ;
            int32_t distance[MAX_AXIS] ;
        } faxis ;
        struct {
            int32_t absolute ;
            int32_t machine ;
            int32_t relative ;
            int32_t distance ;
        } oaxis ;
    } pos ;
} ODBDY2_T ;

typedef struct odbdy2_t_oaxis {
  struct {
      int32_t absolute[MAX_AXIS] ;
      int32_t machine[MAX_AXIS] ;
      int32_t relative[MAX_AXIS] ;
      int32_t distance[MAX_AXIS] ;
  } faxis ;
} ODBDY2_T_OAXIS;

*/
import "C"

import (
	"fmt"
	// 	"go.uber.org/config"
	// 	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"errors"
	"github.com/kr/pretty"
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
		const key = "id"
		last, ok := (*a)[key]
		if !ok || last != value {
			(*b)[key] = value
			*c = append(*c, Update{"id", value})
		}

		return nil
	}
}

func read_parameter(libh C.ushort, key string, num int, ex interface{}) sig {
	return func(a *map[string]interface{}, b *map[string]interface{}, c *[]Update) error {
		var param C.iodbpsd_t

		if ret := C.cnc_rdparam(libh, (C.short)(num), C.ALL_AXES, 8, (*C.IODBPSD)(unsafe.Pointer(&param))); ret != C.EW_OK {
			return errors.New(fmt.Sprintf("cnc_rdparam failed (%d)", ret))
		}
		last, ok := (*a)[key]
		switch ex.(type) {
		case C.long:
			ptr := (*C.long)(unsafe.Pointer(&param.u))
			value := int(*ptr)
			if !ok || last != value {
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

		{
			const key = "actf"
			last, ok := (*a)[key]
			value := int(dyn.actf)
			if !ok || last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		}
		{
			const key = "acts"
			last, ok := (*a)[key]
			value := int(dyn.acts)
			if !ok || last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		}
		{
			const key = "prgmnum"
			last, ok := (*a)[key]
			value := int(dyn.prgmnum)
			if !ok || last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		}
		{
			const key = "prgnum"
			last, ok := (*a)[key]
			value := int(dyn.prgnum)
			if !ok || last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		}
		{
			const key = "seqnum"
			last, ok := (*a)[key]
			value := int(dyn.seqnum)
			if !ok || last != value {
				(*b)[key] = value
				*c = append(*c, Update{key, value})
			}
		}
		{
			oaxis := *(*C.ODBDY2_T_OAXIS)(unsafe.Pointer(&dyn.pos[0]))
			l := C.MAX_AXIS
			if axis_names, ok := (*a)["axis_names"].([]string); ok {
				l = len(axis_names)
			}
			if axis_names, ok := (*b)["axis_names"].([]string); ok {
				l = len(axis_names)
			}
			absolute := make([]int, l)
			distance := make([]int, l)
			machine := make([]int, l)
			relative := make([]int, l)
			for i := 0; i < l; i++ {
				absolute[i] = int(oaxis.faxis.absolute[i])
				distance[i] = int(oaxis.faxis.distance[i])
				machine[i] = int(oaxis.faxis.machine[i])
				relative[i] = int(oaxis.faxis.relative[i])
			}
			{
				last, ok := (*a)["absolute"].([]int)
				for j := 0; ok && j < l; j++ {
					ok = last[j] == absolute[j]
				}
				if !ok {
					(*b)["absolute"] = absolute
					*c = append(*c, Update{"absolute", absolute})
				}
			}
			{
				last, ok := (*a)["distance"].([]int)
				for j := 0; ok && j < l; j++ {
					ok = last[j] == distance[j]
				}
				if !ok {
					(*b)["distance"] = distance
					*c = append(*c, Update{"distance", distance})
				}
			}
			{
				last, ok := (*a)["machine"].([]int)
				for j := 0; ok && j < l; j++ {
					ok = last[j] == machine[j]
				}
				if !ok {
					(*b)["machine"] = machine
					*c = append(*c, Update{"machine", machine})
				}
			}
			{
				last, ok := (*a)["relative"].([]int)
				for j := 0; ok && j < l; j++ {
					ok = last[j] == relative[j]
				}
				if !ok {
					(*b)["relative"] = relative
					*c = append(*c, Update{"relative", relative})
				}
			}
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

	ip := "localhost"
	port := 8193
	_ip := C.CString(ip)
	defer C.free(unsafe.Pointer(_ip))
	_port := (C.ushort)(port)

	log.Printf("connecting to %s:%d...\n", ip, port)
	if ret := C.cnc_allclibhndl3(_ip, _port, 10, &libh); ret != C.EW_OK {
		log.Fatalf("cnc_allclibhndl3 failed (%d)\n", ret)
	}

	// read config
	// loop on interval

	const PART_COUNT_PARAMETER = 6711

	fns := []sig{
		read_id(libh),
		read_parameter(libh, "part_count", PART_COUNT_PARAMETER, C.long(0)),
		read_machine_info(libh),
		read_axis_names(libh),
		read_status(libh),
		read_dynamic(libh),
	}

	a := make(map[string]interface{})
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

	fmt.Printf("a: %# v\n", pretty.Formatter(a))
	fmt.Printf("b: %# v\n", pretty.Formatter(b))
	fmt.Printf("c: %# v\n", pretty.Formatter(c))

	if ret := C.cnc_freelibhndl(libh); ret != C.EW_OK {
		log.Fatalf("cnc_freelibhndl failed (%d)\n", ret)
	}

	if ret := C.cnc_exitprocess(); ret != C.EW_OK {
		log.Fatalf("cnc_exitprocess failed (%d)\n", ret)
	}
}
