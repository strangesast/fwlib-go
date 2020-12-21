package main

/*
#cgo CFLAGS: -I./src
#cgo LDFLAGS: -L./extern/fwlib -lfwlib32 -Wl,-rpath=./extern/fwlib
#include "extern/fwlib/fwlib32.h"
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>

int retrieve_id(char *cnc_id) {
  int ret = 0;
  unsigned short libh;
  uint32_t cnc_ids[4];

  if (cnc_startupprocess(0, "focas.log") != EW_OK) {
    fprintf(stderr, "failed to create required log file!\n");
    return 1;
  }

  const char *ip = "localhost";
  short port = 8193;
  printf("connecting to machine at %s:%d...\n", ip, port);
  if (cnc_allclibhndl3(ip, port, 10, &libh) != EW_OK) {
    fprintf(stderr, "failed to connect to cnc!\n");
    return 1;
  }

  if (cnc_rdcncid(libh, (unsigned long *)cnc_ids) != EW_OK) {
    fprintf(stderr, "failed to read cnc id!\n");
    ret = 1;
    goto cleanup;
  }

  snprintf(cnc_id, 40, "%08x-%08x-%08x-%08x", cnc_ids[0], cnc_ids[1],
           cnc_ids[2], cnc_ids[3]);

cleanup:
  if (cnc_freelibhndl(libh) != EW_OK)
    fprintf(stderr, "failed to free library handle!\n");
  cnc_exitprocess();

  return ret;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func main() {
	s := C.malloc(36)
	ret := C.retrieve_id((*C.char)(s))
	if ret != 0 {
		fmt.Println("failed to retrieve machine id!")
	} else {
		machine_id := C.GoString((*C.char)(s))
		fmt.Printf("machine id: %s\n", machine_id)
	}
	C.free(unsafe.Pointer(s))
}
