//go:build windows && 386

package platform

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

// On 386 SHFILEOPSTRUCTW is packed (pack 1): build it byte by byte.
func shFileOperation(from *uint16) error {
	var buf [30]byte
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 0)                                     // hwnd
	le.PutUint32(buf[4:], foDelete)                              // wFunc
	le.PutUint32(buf[8:], uint32(uintptr(unsafe.Pointer(from)))) // pFrom
	le.PutUint32(buf[12:], 0)                                    // pTo
	le.PutUint16(buf[16:], fofAllowUndo|fofNoConfirmation|fofSilent|fofNoErrorUI)
	// fAnyOperationsAborted (18..21), hNameMappings (22..25), lpszProgressTitle (26..29) left at zero
	ret, _, _ := procSHFileOpertn.Call(uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(from) // the pointer lives only inside buf: keep it alive until the call ends
	return shFileOperationResult(ret, int32(le.Uint32(buf[18:])))
}
