/*

Call python3 functions in fastjob

Depends on "github.com/DataDog/go-python3" with iap_patched features

*/
package fastjobpython3

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
    "log"
    
    fastjob "github.com/iapyeh/fastjob"
	model "github.com/iapyeh/fastjob/model"
	python3 "github.com/iapyeh/go-python3"
)

type TreeRoot = model.TreeRoot
type RouteRegister = model.RouteRegister

type Py3Interpreter struct {
	Router  *Py3Router
	modules map[string]*python3.PyObject //module caches for reload module
}

// py3 is singleton the Py3Interpreter
var py3 *Py3Interpreter
// return a singleton of Py3Interpreter
func NewPy3() *Py3Interpreter {
	if py3 != nil {
		return py3
	}
	py3 = &Py3Interpreter{
		Router:  NewPy3Router(model.Router),
		modules: make(map[string]*python3.PyObject),
	}
	err := Initialize() //definded in helper.go
	if err != nil {
		panic(err)
	}
	log.Println("Py3Interper initialized")


    // This will be called after all of the init() were called
    fastjob.RunInMain(99,func(){
        // Trigger the "callWhenRunning" event handlers
        python3.CallWhenRunning()
    })

	return py3
}
// Alias of NewPy3
func New() *Py3Interpreter {
    return NewPy3()
}

func (py3 *Py3Interpreter) AddTree(tree ...*TreeRoot) {
	for _, t := range tree {
		python3.AddTree(t)
	}
}
func (py3 *Py3Interpreter) ReloadModule(path string) error {
	if m, ok := py3.modules[path]; ok {
		runtime.LockOSThread()
		gil := python3.PyGILState_Ensure()
		defer python3.PyGILState_Release(gil)
		defer runtime.UnlockOSThread()

		importlib := python3.PyImport_ImportModule("importlib")
		//args := python3.PyTuple_New(1)
		//python3.PyTuple_SetItem(args, 0, m)
		fmt.Println("importlib=", importlib, "m=", m)
		newmodule := importlib.CallMethodArgs("reload", m)
		if newmodule == nil {
			err := python3.GetPythonError()
			fmt.Printf("Failed to reload moulde %v\n%v\n", path, err.Error())
			return err
		}
		py3.modules[path] = newmodule
	}
	return nil
}

func (py3 *Py3Interpreter) ImportModule(path string) *python3.PyObject {
	runtime.LockOSThread()
	gil := python3.PyGILState_Ensure()
	defer python3.PyGILState_Release(gil)
	defer runtime.UnlockOSThread()

    log.Println("Import python module",path)

    folder, _ := os.Getwd()
    
    abspath := filepath.Join(folder,path)
    _, err := os.Stat(abspath)
    if os.IsNotExist(err) {
        panic(fmt.Sprintf("%v (%v) is not found", abspath))
    }    

	modulename := strings.TrimSuffix(path, filepath.Ext(filepath.Base(path)))
    dotmodulename := strings.Join(strings.Split(modulename,"/"),".")
	sys := python3.PyImport_GetModule(python3.PyUnicode_FromString("sys"))
	if sys == nil {
		sys = python3.PyImport_ImportModule("sys")
	}
	syspath := sys.GetAttrString("path")
	var existed = false
	for i := 0; i < python3.PyList_Size(syspath); i++ {
		p := python3.PyUnicode_AsUTF8(python3.PyList_GetItem(syspath, i))
		if p == folder {
			existed = true
			break
		}
	}
	if !existed {
		//insert folder into sys.path
		python3.PyList_Insert(syspath, 0, python3.PyUnicode_FromString(folder))
	}


	importlib := python3.PyImport_GetModule(python3.PyUnicode_FromString("importlib"))
	if importlib == nil {
		importlib = python3.PyImport_ImportModule("importlib")
	}
    importmodule := importlib.GetAttrString("import_module")
	if importmodule == nil {
		panic("can not load import_module")
	}
    args := python3.PyTuple_New(1)
	python3.PyTuple_SetItem(args, 0, python3.PyUnicode_FromString(dotmodulename) )
	module := importmodule.CallObject(args)

	if module == nil {
		err := python3.GetPythonError()
		panic(fmt.Errorf("Failed to import \"%v\" (module %s) from folder %v\nError:%v", path, modulename, folder, err.Error()))
	}

	py3.modules[path] = module
	return module
}
/*
func (py3 *Py3Interpreter) OLDImportModule(path string) *python3.PyObject {
	runtime.LockOSThread()
	gil := python3.PyGILState_Ensure()
	defer python3.PyGILState_Release(gil)
	defer runtime.UnlockOSThread()

	folder := filepath.Dir(path)
	if folder == "" {
		folder, _ = os.Getwd()
	}
    
    folder, err := filepath.Abs(folder)
    if err != nil{
        panic(err)
    }

    _, err = os.Stat(folder)
    if os.IsNotExist(err) {
        panic(fmt.Sprintf("%v (%v) is not found", folder))
    }    

    abspath := filepath.Join(folder,filepath.Base(path))
    _, err = os.Stat(abspath)
    if os.IsNotExist(err) {
        panic(fmt.Sprintf("%v (%v) is not found", abspath))
    }    

	filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
	var sys *python3.PyObject

	sys = python3.PyImport_GetModule(python3.PyUnicode_FromString("sys"))
	if sys == nil {
		sys = python3.PyImport_ImportModule("sys")
	}
	syspath := sys.GetAttrString("path")
	var existed = false
	for i := 0; i < python3.PyList_Size(syspath); i++ {
		p := python3.PyUnicode_AsUTF8(python3.PyList_GetItem(syspath, i))
		if p == folder {
			existed = true
			break
		}
	}
	if !existed {
		//insert folder into sys.path
		python3.PyList_Insert(syspath, 0, python3.PyUnicode_FromString(folder))
	}
	module := python3.PyImport_ImportModule(filename)
	if module == nil {
		err := python3.GetPythonError()
		panic(fmt.Errorf("Failed to import moulde %v from %v\n%v", filename, folder, err.Error()))
	}

	if !existed {
		// 2019-09-15T13:06:19+00:00 不要移除掉，因為這樣才能reload module
		//sys.SetAttrString("path",
		//	python3.PyList_GetSlice(syspath, 1, python3.PyList_Size(syspath)))
	}

	defer sys.DecRef()
	defer syspath.DecRef()

	py3.modules[path] = module
	return module
}
*/

type Py3Router struct {
	Router *RouteRegister
}

var singletonPy3Route *Py3Router

func NewPy3Router(r *RouteRegister) *Py3Router {
	if singletonPy3Route == nil {
		singletonPy3Route = &Py3Router{
			Router: r, //objsh.Router,
		}
	}
	return singletonPy3Route
}
