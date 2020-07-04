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
	mask := ^(unix.ECHO | unix.ICANON)
	termios.Lflag &= uint32(mask)

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, termios); err != nil {
		panic(err)
	}
}

func disableRawMode(origTermios *unix.Termios) {
	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, origTermios); err != nil {
		panic(err)
	}
}

func main() {
	origTermios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	enableRawMode()
	defer disableRawMode(origTermios)

	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err == io.EOF || buf[0] == 'q' {
			break
		}
	}
}
