// +build darwin freebsd linux

package daemon

import "syscall"

const (
	SIGUSR1 = syscall.SIGUSR1
	SIGUSR2 = syscall.SIGUSR2

	LOCK_EX = syscall.LOCK_EX
	LOCK_NB = syscall.LOCK_NB
)

func Flock(fd int, how int) error {
	return syscall.Flock(fd, how)
}
