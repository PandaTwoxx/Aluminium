package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

type UserCreateInput struct {
	Username string
	Password string
	Email    string
}

func UserCreate(defaults UserCreateInput) (*UserCreateInput, error) {
	result := defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Description("Choose a username").
				Value(&result.Username).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("username is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				Description("Choose a password").
				EchoMode(huh.EchoModePassword).
				Value(&result.Password).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("password is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Email").
				Description("Your email address").
				Value(&result.Email).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("email is required")
					}
					return nil
				}),
		).Title("Create User").
			Description("Create a new user on the server"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &result, nil
}
