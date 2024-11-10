package prompter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/matthewmueller/diff"
	"github.com/matthewmueller/prompter"
)

func TestAsk(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n27\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "27")
}
func TestAskErrRequired(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n27\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "27")
	height, err := prompt.Ask(ctx, "What is your height?")
	is.True(errors.Is(err, prompter.ErrRequired))
	is.Equal(height, "")
}

func TestAskOptional(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Optional(true).Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "")
}

func TestAskDefault(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Default("21").Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "21")
}

func TestAskValidate(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	writer := new(bytes.Buffer)
	reader := io.NopCloser(bytes.NewBufferString("Am\nAmy\n"))
	prompt := prompter.New(writer, reader)
	validName := func(s string) error {
		if len(s) < 3 {
			return fmt.Errorf("'%s' is too short", s)
		}
		return nil
	}
	name, err := prompt.Is(validName).Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Amy")
	diff.TestString(t, writer.String(), "What is your name? 'Am' is too short\nWhat is your name? ")
}

func TestAskDefaultGiven(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n27\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Default("21").Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "27")
}

func TestAskDefaultOptional(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Optional(true).Default("21").Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "21")
}

func TestAskDefaultOptionalGiven(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n27\n"))
	prompt := prompter.New(os.Stdout, reader)
	name, err := prompt.Ask(ctx, "What is your name?")
	is.NoErr(err)
	is.Equal(name, "Mark")
	age, err := prompt.Optional(true).Default("21").Ask(ctx, "What is your age?")
	is.NoErr(err)
	is.Equal(age, "27")
}

func TestPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("some password\n"))
	prompt := prompter.New(os.Stdout, reader)
	pass, err := prompt.Password(ctx, "What is your password?")
	is.NoErr(err)
	is.Equal(pass, "some password")
}

func TestPasswordDefault(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString(""))
	prompt := prompter.New(os.Stdout, reader)
	pass, err := prompt.Default("idk").Password(ctx, "What is your password?")
	is.NoErr(err)
	is.Equal(pass, "idk")
}

func TestPasswordOptional(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString(""))
	prompt := prompter.New(os.Stdout, reader)
	pass, err := prompt.Optional(true).Password(ctx, "What is your password?")
	is.NoErr(err)
	is.Equal(pass, "")
}

func TestPasswordValidate(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("mypassword\nsome password\n"))
	prompt := prompter.New(os.Stdout, reader)
	validate := func(s string) error {
		if s != "some password" {
			return errors.New("invalid password")
		}
		return nil
	}
	pass, err := prompt.Is(validate).Password(ctx, "What is your password?")
	is.NoErr(err)
	is.Equal(pass, "some password")
}

func TestConfirmTrue(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("hello\nyes\n"))
	prompt := prompter.New(os.Stdout, reader)
	create, err := prompt.Confirm(ctx, "Create new user? (yes/no)")
	is.NoErr(err)
	is.Equal(create, true)
}

func TestConfirmFalse(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	reader := io.NopCloser(bytes.NewBufferString("hello\nno\n"))
	prompt := prompter.New(os.Stdout, reader)
	create, err := prompt.Confirm(ctx, "Create new user? (yes/no)")
	is.NoErr(err)
	is.Equal(create, false)
}

func TestAskCancel(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := io.NopCloser(bytes.NewBufferString("Mark\n"))
	prompt := prompter.New(os.Stdout, reader)
	cancel() // Cancel the context before asking
	name, err := prompt.Ask(ctx, "What is your name?")
	is.True(errors.Is(err, context.Canceled))
	is.Equal(name, "")
}

// func TestPasswordCancel(t *testing.T) {
// 	is := is.New(t)
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	reader := io.NopCloser(bytes.NewBufferString("some password\n"))
// 	prompt := prompter.New(os.Stdout, reader)
// 	cancel() // Cancel the context before asking
// 	_, err := prompt.Password(ctx, "What is your password?")
// 	is.True(errors.Is(err, context.Canceled))
// }

func TestConfirmCancel(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := io.NopCloser(bytes.NewBufferString("yes\n"))
	prompt := prompter.New(os.Stdout, reader)
	cancel() // Cancel the context before asking
	_, err := prompt.Confirm(ctx, "Create new user? (yes/no)")
	is.True(errors.Is(err, context.Canceled))
}
