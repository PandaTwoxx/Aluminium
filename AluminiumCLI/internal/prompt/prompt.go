package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

// SimpleConfirm prompts with a plain [y/N] line when huh is unavailable.
func SimpleConfirm(message string, defaultNo bool) (bool, error) {
	defaultHint := "y/N"
	if !defaultNo {
		defaultHint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", message, defaultHint)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultNo, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return !defaultNo, nil
	}
	return input == "y" || input == "yes", nil
}

// Confirm shows a styled yes/no prompt.
func Confirm(title string, defaultYes bool) (bool, error) {
	var result bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&result).
		Run()
	if err != nil {
		return defaultYes, err
	}
	return result, nil
}
