// +build linux

package fds

import "io/ioutil"

const fdsDirPath = "/proc/self/fd"

// GetOpenFds returnds a number of opened fds by the process
func GetOpenFds() (int, error) {
	dirs, err := ioutil.ReadDir(fdsDirPath)
	if err != nil {
		return 0, err
	}

	return len(dirs), nil
}
