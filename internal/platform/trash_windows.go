package platform

import (
	"errors"
	"fmt"
	"syscall"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procSHFileOpertn = shell32.NewProc("SHFileOperationW")
)

// shFileOpStruct is SHFILEOPSTRUCTW. On 64 bit shellapi.h uses natural
// alignment, which is also Go's; on 386 it uses pack(1), see trash_windows_386.go.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

const (
	foDelete          = 0x0003
	fofSilent         = 0x0004
	fofNoConfirmation = 0x0010
	fofAllowUndo      = 0x0040
	fofNoErrorUI      = 0x0400
)

// MoveToTrash moves path to the trash (it can be restored from there).
func MoveToTrash(path string) error {
	from, err := syscall.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0) // the list must end with a double NUL
	return shFileOperation(&from[0])
}

func shFileOperationResult(ret uintptr, aborted int32) error {
	if aborted != 0 {
		return errors.New("operation aborted")
	}
	if ret != 0 {
		return shError(ret)
	}
	return nil
}

// shError is a SHFileOperation return code (the DE_* values of shellapi.h, or a Win32 error).
type shError uintptr

func (e shError) Error() string { return fmt.Sprintf("SHFileOperation failed (code 0x%x)", uintptr(e)) }

// inUse: what SHFileOperation returns when another program keeps a file or the folder open
// (an editor, a terminal whose working directory is inside it…): DE_ACCESSDENIEDSRC (0x78),
// DE_INVALIDFILES (0x7C, despite the name: it is what a locked file gives with FOF_NOERRORUI),
// or the Win32 access denied, sharing and lock violations.
func (e shError) inUse() bool {
	switch e {
	case 0x78, 0x7C, 5, 32, 33:
		return true
	}
	return false
}
