package prompter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrRequired is returned when a required input is empty
var ErrRequired = fmt.Errorf("prompter: input is required")

// Default creates a default prompter using stdin and stdout
func Default() *Prompter {
	return New(os.Stdout, os.Stdin)
}

// New created a default prompter
func New(w io.Writer, r io.Reader) *Prompter {
	fd := getFd(r)
	return &Prompter{
		writer: w,
		reader: bufio.NewReader(r),
		fd:     fd,
	}
}

type fd interface {
	Fd() uintptr
}

func getFd(r io.Reader) int {
	if f, ok := r.(fd); ok {
		return int(f.Fd())
	}
	return -1
}

// Prompter can ask for inputs and validate them
type Prompter struct {
	writer io.Writer
	reader *bufio.Reader
	fd     int
}

// Default sets the default value for the question
func (p *Prompter) Default(defaultTo string) *Question {
	q := newQuestion(p)
	q.defaultTo = defaultTo
	return q
}

// Optional sets the question as optional
func (p *Prompter) Optional(optional bool) *Question {
	q := newQuestion(p)
	q.optional = optional
	return q
}

// Is adds validators to the question
func (p *Prompter) Is(validators ...func(string) error) *Question {
	q := newQuestion(p)
	q.validators = append(q.validators, validators...)
	return q
}

// Ask asks a question and returns the input
func (p *Prompter) Ask(ctx context.Context, prompt string) (string, error) {
	q := newQuestion(p)
	return q.Ask(ctx, prompt)
}

// Password asks for a password and returns the input
func (p *Prompter) Password(ctx context.Context, prompt string) (string, error) {
	q := newQuestion(p)
	return q.Password(ctx, prompt)
}

// Confirm asks for a confirmation and returns the input
func (p *Prompter) Confirm(ctx context.Context, prompt string) (bool, error) {
	q := newQuestion(p)
	return q.Confirm(ctx, prompt)
}

func newQuestion(p *Prompter) *Question {
	return &Question{
		prompter: p,
	}
}

// Question that can be asked
type Question struct {
	prompter   *Prompter
	validators []func(string) error
	defaultTo  string
	optional   bool
}

func (q *Question) scanLine(inputCh chan<- string, errorCh chan<- error) {
	p := q.prompter

	// Read the input
	input, err := p.reader.ReadString('\n')
	if err != nil {
		if !errors.Is(err, io.EOF) {
			errorCh <- err
			return
		}
		// If we're at the end of the input, and there is a default, use it,
		// otherwise return a required error
		if q.defaultTo != "" {
			inputCh <- q.defaultTo
			return
		} else if !q.optional {
			errorCh <- ErrRequired
			return
		}
	}

	// Trim the input
	input = strings.TrimRight(input, "\r\n")
	inputCh <- input
}

// Read the password. If the file descriptor is available, use term.ReadPassword
// otherwise read the line from the scanner
func (q *Question) scanPassword(inputCh chan<- string, errorCh chan<- error) {
	p := q.prompter

	if p.fd > -1 && term.IsTerminal(p.fd) {
		pass, err := term.ReadPassword(p.fd)
		if err != nil {
			errorCh <- err
			return
		}
		inputCh <- string(pass)
		return
	}

	q.scanLine(inputCh, errorCh)
}

// Default sets the default value for the question
func (q *Question) Default(defaultTo string) *Question {
	q.defaultTo = defaultTo
	return q
}

// Optional sets the question as optional
func (q *Question) Optional(optional bool) *Question {
	q.optional = optional
	return q
}

// Is adds validators to the question
func (q *Question) Is(validators ...func(string) error) *Question {
	q.validators = append(q.validators, validators...)
	return q
}

// Reads the input from the reader
func (q *Question) readInput(ctx context.Context) (string, error) {
	// Check if the context has already been cancelled
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	inputCh := make(chan string)
	errorCh := make(chan error)

	// Scan for the input in a goroutine, so we can listen for cancellations.
	go q.scanLine(inputCh, errorCh)

	// Wait for input, an error or the context to be cancelled
	select {
	case input := <-inputCh:
		close(inputCh)
		close(errorCh)
		return input, nil
	case err := <-errorCh:
		close(inputCh)
		close(errorCh)
		return "", err
	case <-ctx.Done():
		// In this case, we're leaking the goroutine that's reading the input.
		// This is because we can't really cancel reads without limitations.
		// This seems acceptable because typically when context is canceled, the
		// process will exit shortly.
		return "", ctx.Err()
	}
}

// Reads the password from the reader
func (q *Question) readPassword(ctx context.Context) (string, error) {
	// Check if the context has already been cancelled
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	inputCh := make(chan string)
	errorCh := make(chan error)

	// Scan for the password in a goroutine, so we can listen for cancelations.
	go q.scanPassword(inputCh, errorCh)

	// Wait for input, an error or the context to be cancelled
	select {
	case input := <-inputCh:
		close(inputCh)
		close(errorCh)
		return input, nil
	case err := <-errorCh:
		close(inputCh)
		close(errorCh)
		return "", err
	case <-ctx.Done():
		// In this case, we're leaking the goroutine that's reading the password.
		// This is because we can't really cancel reads without limitations.
		// This seems acceptable because typically when context is canceled, the
		// process will exit shortly.
		return "", ctx.Err()
	}
}

// Ask asks a question and returns the input
func (q *Question) Ask(ctx context.Context, prompt string) (string, error) {
	p := q.prompter

	// Write out the formatted prompt
retry:
	fmt.Fprint(p.writer, prompt, " ")

	// Read the input
	input, err := q.readInput(ctx)
	if err != nil {
		return "", err
	}

	// If the input is empty, and there is a default, use it otherwise ask again
	if input == "" {
		if q.defaultTo != "" {
			return q.defaultTo, nil
		} else if !q.optional {
			goto retry
		}
	}

	// If any validators fail, print the error and ask again
	for _, validate := range q.validators {
		if err := validate(input); err != nil {
			fmt.Fprintln(p.writer, err)
			goto retry
		}
	}

	return input, nil
}

// Password asks for a password and returns the input
func (q *Question) Password(ctx context.Context, prompt string) (string, error) {
	p := q.prompter

	// Write out the formatted prompt
retry:
	fmt.Fprint(p.writer, prompt, " ")

	// Read the input
	pass, err := q.readPassword(ctx)
	if err != nil {
		return "", err
	}
	// Print a newline after the password
	fmt.Fprintln(p.writer)

	if pass == "" {
		if q.defaultTo != "" {
			return q.defaultTo, nil
		} else if !q.optional {
			goto retry
		}
	}

	// If any validators fail, print the error and ask again
	for _, validate := range q.validators {
		if err := validate(pass); err != nil {
			fmt.Fprintln(p.writer, err)
			goto retry
		}
	}

	return pass, nil
}

func isYes(s string) bool {
	switch strings.ToLower(s) {
	case "y", "yes", "true":
		return true
	}
	return false
}

// Confirm asks for a confirmation and returns the input
func (q *Question) Confirm(ctx context.Context, prompt string) (bool, error) {
	// Add a validator to ensure the input is yes or no
	q.validators = append(q.validators, func(s string) error {
		switch strings.ToLower(s) {
		case "y", "yes":
			return nil
		case "n", "no":
			return nil
		default:
			return fmt.Errorf("invalid value %q, must enter yes or no", s)
		}
	})

	input, err := q.Ask(ctx, prompt)
	if err != nil {
		return false, err
	}

	return isYes(input), nil
}
