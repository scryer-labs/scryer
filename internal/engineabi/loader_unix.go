//go:build cgo && (darwin || linux)

package engineabi

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include "scryer_engine_v1.h"

typedef scryer_result_v1 (*execute_fn)(scryer_view_v1);
typedef bool (*cancel_fn)(scryer_view_v1);
typedef void (*release_fn)(scryer_buffer_v1);

static scryer_result_v1 call_execute(void *fn, const char *data, size_t size) {
    return ((execute_fn)fn)((scryer_view_v1){data, size});
}

static bool call_cancel(void *fn, const char *data, size_t size) {
    return ((cancel_fn)fn)((scryer_view_v1){data, size});
}

static void call_release(void *fn, scryer_buffer_v1 buffer) {
    ((release_fn)fn)(buffer);
}

static void *open_engine(const char *path, char **error) {
    void *handle = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (!handle) {
        const char *message = dlerror();
        if (message) *error = strdup(message);
    }
    return handle;
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

type library struct {
	executeFn, cancelFn, releaseFn unsafe.Pointer
}

var loaded struct {
	sync.Mutex
	libraries map[string]library
}

func open(path string) (library, error) {
	loaded.Lock()
	defer loaded.Unlock()
	if existing, ok := loaded.libraries[path]; ok {
		return existing, nil
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var detail *C.char
	handle := C.open_engine(cpath, &detail)
	if handle == nil {
		if detail == nil {
			return library{}, fmt.Errorf("load engine %q", path)
		}
		defer C.free(unsafe.Pointer(detail))
		return library{}, fmt.Errorf("load engine %q: %s", path, C.GoString(detail))
	}
	// A shared library may contain a live language runtime. Keep it loaded until process exit.
	lookup := func(symbol string) unsafe.Pointer {
		name := C.CString(symbol)
		defer C.free(unsafe.Pointer(name))
		return C.dlsym(handle, name)
	}
	result := library{lookup("execute_v1"), lookup("cancel_v1"), lookup("release_buffer_v1")}
	if result.executeFn == nil || result.cancelFn == nil || result.releaseFn == nil {
		return library{}, fmt.Errorf("engine %q is missing required ABI v1 symbols", path)
	}
	if loaded.libraries == nil {
		loaded.libraries = make(map[string]library)
	}
	loaded.libraries[path] = result
	return result, nil
}

func (l library) execute(request []byte) ([]byte, int32, error) {
	input := C.CBytes(request)
	defer C.free(input)
	result := C.call_execute(l.executeFn, (*C.char)(input), C.size_t(len(request)))
	buffer := result.response
	if buffer.data != nil {
		defer C.call_release(l.releaseFn, buffer)
	}
	if buffer.size > 256*1024*1024 || (buffer.data == nil && buffer.size != 0) {
		return nil, int32(result.status), fmt.Errorf("invalid engine ABI response buffer")
	}
	return C.GoBytes(unsafe.Pointer(buffer.data), C.int(buffer.size)), int32(result.status), nil
}

func (l library) cancel(id string) bool {
	input := C.CString(id)
	defer C.free(unsafe.Pointer(input))
	return bool(C.call_cancel(l.cancelFn, input, C.size_t(len(id))))
}
