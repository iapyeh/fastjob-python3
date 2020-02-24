/*
Python3 interpreter control
*/

package fastjobpython3

import (
	model "github.com/iapyeh/fastjob/model"
)

type PythonBranch struct {
	model.BaseBranch
}

//var emptyArray = make([]int8, 0)

func (pybranch *PythonBranch) BeReady(treeroot *model.TreeRoot) {
	pybranch.SetName("$py")
	pybranch.InitBaseBranch()
	pybranch.Export(
		pybranch.ReloadModule,
	)
	treeroot.SureReady(pybranch)
}

/* ReloadModule reload python module
Args:[path]
*/
func (pybranch *PythonBranch) ReloadModule(ctx *model.TreeCallCtx) {

	path := ctx.Args[0]
	py3 := NewPy3()
	err := py3.ReloadModule(path)
	if err == nil {
		ctx.Resolve("OK")
	} else {
		ctx.Reject(500, err)
	}

}
