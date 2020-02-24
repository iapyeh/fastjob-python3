// 2019-08-19T14:16:32+00:00
// Rewrite from codes bollowed from wsgi.go
package fastjobpython3

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"

	python3 "github.com/iapyeh/go-python3"
)

/*
#cgo !windows pkg-config: python-3.7
#include "helper.h"
*/
import "C"

type PyObject C.PyObject

var pythonError = python3.GetPythonError

var (
	//contexts []context // Storage for active request data.
	//slots    chan int  // Worker thread pool to limit the number of Python threads.
	//wsgiVersion     *C.PyObject
	managerOut      chan error
	managerFinalize chan int
	// PyStringType works around a cgo Windows bug where data
	// exported from a DLL cannot be referenced directly by Go.
	// We get the address of PyString_Type in C at initialization
	// and return it to go via a function.
	// See https://github.com/golang/go/issues/4339
	//pyStringType *C.PyTypeObject
	iomodule *python3.PyObject
)

func pyUnicode_ToGoString(s *C.PyObject) string {
	cutf8 := C.PyUnicode_AsUTF8(s)
	return C.GoString(cutf8)
}
func pyUnicode_FromGoString(u string) *C.PyObject {
	cu := C.CString(u)
	defer C.free(unsafe.Pointer(cu))
	return (*C.PyObject)(C.PyUnicode_FromString(cu))
}

func Togo(cobject *C.PyObject) *PyObject {
	return (*PyObject)(cobject)
}

// Initialize sets up the embedded Python interpreter.
// Note that Python initialization errors will often
// cause the program to exit, rather than returning an
// error here. This is because the embedded Python library
// calls exit(3) or abort(3) on most initialization
// errors.
func Initialize() error {
	managerOut = make(chan error)
	managerFinalize = make(chan int)
	go manager()
	return <-managerOut
}

// Finalize tears down the embedded Python interpreter. Do not call if
// Initialize returned an error.
func Finalize() {
	managerFinalize <- 1
	<-managerOut
}

// Manager manages initialization and finalization of a Python interpreter. It
// runs as a goroutine locked to a single OS thread, since Python needs to be
// initialized and finalized from the same thread.
func manager() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := initializePython()
	if err != nil {
		log.Fatalf("Initialize Python Error: %v", err)
		managerOut <- err
		return
	}

	// 2019-11-17T05:01:25+00:00 The following statement maybe obsoleted?
	// 		If the threading module is ever imported, it must be imported from
	// 		this thread. Otherwise, Python complains.
	threading := python3.PyImport_ImportModule("threading")
	if threading == nil {
		err := pythonError()
		C.Py_Finalize()
		managerOut <- err
		return
	}

	//initialize fastjob/objsh related python  modules
	python3.InitIapPatchedModules()

	ts := C.PyEval_SaveThread()
	managerOut <- nil
	<-managerFinalize
	C.PyEval_RestoreThread(ts)

	C.Py_Finalize()
	managerOut <- nil
}

func initializePython() error {
	var err error
	if C.Py_IsInitialized() != 0 {
		return fmt.Errorf("wsgi: Python already initialized")
	}

	IgnoreEnvironment := true

	if IgnoreEnvironment {
		C.Py_IgnoreEnvironmentFlag = 1
	} else {
		C.Py_IgnoreEnvironmentFlag = 0
	}

	C.Py_InitializeEx(0)
	C.PyEval_InitThreads()
	if C.Py_IsInitialized() == 0 {
		return fmt.Errorf("wsgi: Couldn't initialize Python interpreter.")
	}

	return err
}

// CallWhenRunning , 現在是手動呼叫，將來應該放在AddTree時，在tree ready時自動呼叫
// (2019-10-23T14:45:51+00:00)
func CallWhenRunning() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	go python3.CallWhenRunning()
}
