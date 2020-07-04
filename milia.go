package main

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func enableRawMode() {
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	mask := ^(unix.ECHO)
	termios.Lflag &= uint32(mask)

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, termios); err != nil {
		panic(err)
	}
}

func disableRawMode() {
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	termios.Lflag |= unix.ECHO

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, termios); err != nil {
		panic(err)
	}
}

func main() {
	enableRawMode()
	defer disableRawMode()

	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err == io.EOF || buf[0] == 'q' {
			break
		}
	}
}
