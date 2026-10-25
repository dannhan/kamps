//go:build windows

package library

import "golang.org/x/sys/windows"

// availableBytes returns filesystem bytes available to the current user.
func availableBytes(path string) (int64, error) {
	dir, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var free, total, available uint64
	if err := windows.GetDiskFreeSpaceEx(dir, &free, &total, &available); err != nil {
		return 0, err
	}
	return int64(available), nil
}
