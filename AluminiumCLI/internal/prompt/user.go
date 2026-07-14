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

type LoginInput struct {
	Username string
	Password string
}

func Login(defaults LoginInput) (*LoginInput, error) {
	result := defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Description("Your username").
				Value(&result.Username).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("username is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				Description("Your password").
				EchoMode(huh.EchoModePassword).
				Value(&result.Password).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("password is required")
					}
					return nil
				}),
		).Title("Login").
			Description("Log in to generate a token"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &result, nil
}

type TokenRevokeInput struct {
	Token    string
	Password string
}

func TokenRevoke(defaults TokenRevokeInput) (*TokenRevokeInput, error) {
	result := defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Token").
				Description("Token to revoke").
				Value(&result.Token).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("token is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Password").
				Description("User password").
				EchoMode(huh.EchoModePassword).
				Value(&result.Password).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("password is required")
					}
					return nil
				}),
		).Title("Revoke Token"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &result, nil
}

type TokenValidateInput struct {
	Token string
}

func TokenValidate(defaults TokenValidateInput) (*TokenValidateInput, error) {
	result := defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Token").
				Description("Token to validate").
				Value(&result.Token).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("token is required")
					}
					return nil
				}),
		).Title("Validate Token"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &result, nil
}

type ScopeGrantInput struct {
	Username string
	Scope    string
}

func ScopeGrant(defaults ScopeGrantInput) (*ScopeGrantInput, error) {
	result := defaults

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Description("Username of the target user").
				Value(&result.Username).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("username is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Scope").
				Description("Scope to grant (e.g. read, write, admin)").
				Value(&result.Scope).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("scope is required")
					}
					return nil
				}),
		).Title("Grant Scope"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &result, nil
}

func ChooseScopes(userScopes []string) ([]string, error) {
	allScopes := []string{"read", "write", "dev", "admin"}
	var options []huh.Option[string]

	for _, scope := range allScopes {
		has := false
		for _, s := range userScopes {
			if s == scope {
				has = true
				break
			}
		}
		if has {
			options = append(options, huh.NewOption(scope, scope))
		} else {
			// Grey out using faint ANSI escape code
			label := fmt.Sprintf("\x1b[2m%s (unavailable)\x1b[0m", scope)
			options = append(options, huh.NewOption(label, scope+"_unavailable"))
		}
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Scopes").
				Description("Choose the scopes to grant to this token").
				Options(options...).
				Value(&selected).
				Validate(func(val []string) error {
					for _, v := range val {
						if strings.HasSuffix(v, "_unavailable") {
							return fmt.Errorf("cannot select unavailable scope: %s", strings.TrimSuffix(v, "_unavailable"))
						}
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
