# Prompter

**Deprecated:** use https://github.com/Bowery/prompt instead.

---

[![Go Reference](https://pkg.go.dev/badge/github.com/matthewmueller/prompter.svg)](https://pkg.go.dev/github.com/matthewmueller/prompter)

Minimal prompting library for Go.

## Features

- Fluent API
- Supports inputs, passwords and confirmations
- Supports validations, defaults and optionals
- Supports context canceling

## Install

```sh
go get github.com/matthewmueller/prompter
```

## Examples

```go
prompt := prompter.Default()

// Ask for some input
name, err := prompt.Ask("What is your name?")

// Optional inputs
age, err := prompt.Optional(true).Ask("What is your age?")

// Default values
age, err = prompt.Default("21").Ask("What is your age?")

// Validations
func validPass(input string) error {
  if len(input) < 8 {
    return errors.New("password is too short")
  }
}

// Passwords
pass, err := prompt.Is(validPass).Password("What is your password?")

// Confirmations
shouldCreate, err := prompt.Confirm("Create new user? (yes/no)")

// Chaining
func validAge(input string) error {
  n, err := strconv.Atoi(input)
  if err != nil {
    return fmt.Errorf("%q must be a number")
  } else if n < 0 {
    return fmt.Errorf("%q must be greater than 0")
  }
  return nil
}
age, err := prompt.Default("21").Is(validAge).Ask("What is your age?")
```

## Development

First, clone the repo:

```sh
git clone https://github.com/matthewmueller/prompter
cd prompter
```

Next, install dependencies:

```sh
go mod tidy
```

Finally, try running the tests:

```sh
go test ./...
```

## License

MIT
