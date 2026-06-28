package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Prompt asks the user for a text input.
func Prompt(in io.Reader, out io.Writer, msg string) (string, error) {
	fmt.Fprintf(out, "%s ", msg)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// PromptPassword asks the user for a sensitive input (hides typing).
func PromptPassword(in io.Reader, out io.Writer, msg string) (string, error) {
	fmt.Fprintf(out, "%s ", msg)

	// term.ReadPassword expects a file descriptor for stdin (usually 0).
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Fprintln(out) // Add newline after password entry
	return strings.TrimSpace(string(bytePassword)), nil
}
