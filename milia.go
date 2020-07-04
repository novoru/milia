package main

import (
	"fmt"
	"io"
	"os"
	"unicode"

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

func disableRawMode() {
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	termios.Lflag |= uint32(unix.ECHO | unix.ICANON)

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
		if unicode.IsControl(rune(buf[0])) {
			fmt.Printf("%d\n", buf[0])
		} else {
			fmt.Printf("%d ('%c')\n", buf[0], buf[0])
		}
	}
}
