//go:build windows && !386

package platform

import "unsafe"

func shFileOperation(from *uint16) error {
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  from,
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	ret, _, _ := procSHFileOpertn.Call(uintptr(unsafe.Pointer(&op)))
	return shFileOperationResult(ret, op.fAnyOperationsAborted)
}
