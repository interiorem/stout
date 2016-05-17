// +build darwin

package fds

import "path/filepath"

// `fdesc` is typically mounted to `/dev`
const fdsDirPathMask = "/dev/fd/*"

func GetOpenFds() (int, error) {
	dirs, err := filepath.Glob(fdsDirPathMask)
	if err != nil {
		return 0, err
	}

	return len(dirs), nil
}
