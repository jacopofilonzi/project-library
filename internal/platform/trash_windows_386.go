//go:build windows && 386

package platform

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

// Su 386 SHFILEOPSTRUCTW è impacchettata (pack 1): la costruisco byte per byte.
func shFileOperation(from *uint16) error {
	var buf [30]byte
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 0)                                     // hwnd
	le.PutUint32(buf[4:], foDelete)                              // wFunc
	le.PutUint32(buf[8:], uint32(uintptr(unsafe.Pointer(from)))) // pFrom
	le.PutUint32(buf[12:], 0)                                    // pTo
	le.PutUint16(buf[16:], fofAllowUndo|fofNoConfirmation|fofSilent|fofNoErrorUI)
	// fAnyOperationsAborted (18..21), hNameMappings (22..25), lpszProgressTitle (26..29) a zero
	ret, _, _ := procSHFileOpertn.Call(uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(from) // il puntatore vive solo dentro buf: va tenuto vivo fino alla fine della chiamata
	return shFileOperationResult(ret, int32(le.Uint32(buf[18:])))
}
