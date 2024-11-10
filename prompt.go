package prompter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

var ErrRequired = fmt.Errorf("prompter: input is required")

func Default() *Prompter {
	return New(os.Stdout, os.Stdin)
}

// New created a default prompter
func New(w io.Writer, r io.Reader) *Prompter {
	fd := getFd(r)
	cr, _ := cancelreader.NewReader(r)
	return &Prompter{
		writer:  w,
		scanner: bufio.NewScanner(cr),
		fd:      fd,
		cancel:  cr.Cancel,
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

type Prompter struct {
	writer  io.Writer
	scanner *bufio.Scanner
	fd      int
	cancel  func() bool
}

func (p *Prompter) Default(defaultTo string) *Question {
	q := newQuestion(p)
	q.defaultTo = defaultTo
	return q
}

func (p *Prompter) Optional(optional bool) *Question {
	q := newQuestion(p)
	q.optional = optional
	return q
}

func (p *Prompter) Is(validators ...func(string) error) *Question {
	q := newQuestion(p)
	q.validators = append(q.validators, validators...)
	return q
}

func (p *Prompter) Ask(ctx context.Context, prompt string) (string, error) {
	q := newQuestion(p)
	return q.Ask(ctx, prompt)
}

func (p *Prompter) Password(ctx context.Context, prompt string) (string, error) {
	q := newQuestion(p)
	return q.Password(ctx, prompt)
}

func (p *Prompter) Confirm(ctx context.Context, prompt string) (bool, error) {
	q := newQuestion(p)
	return q.Confirm(ctx, prompt)
}

func newQuestion(p *Prompter) *Question {
	return &Question{
		prompter: p,
	}
}

type Question struct {
	prompter   *Prompter
	validators []func(string) error
	defaultTo  string
	optional   bool
}

func (q *Question) scanLine(inputCh chan<- string, errorCh chan<- error) {
	p := q.prompter

	// Read the input
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
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
	input := strings.TrimRight(p.scanner.Text(), "\r\n")
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

func (q *Question) Default(defaultTo string) *Question {
	q.defaultTo = defaultTo
	return q
}

func (q *Question) Optional(optional bool) *Question {
	q.optional = optional
	return q
}

func (q *Question) Is(validators ...func(string) error) *Question {
	q.validators = append(q.validators, validators...)
	return q
}

func (q *Question) readInput(ctx context.Context) (string, error) {
	inputCh := make(chan string)
	defer close(inputCh)
	errorCh := make(chan error)
	defer close(errorCh)

	go q.scanLine(inputCh, errorCh)

	select {
	case input := <-inputCh:
		return input, nil
	case err := <-errorCh:
		return "", err
	case <-ctx.Done():
		// Cancel the underlying reader
		q.prompter.cancel()

		// Wait for the loop to exit
		// TODO: this can hang
		<-errorCh

		// Return the canceled error
		return "", ctx.Err()
	}
}

func (q *Question) readPassword(ctx context.Context) (string, error) {
	inputCh := make(chan string)
	defer func() {
		close(inputCh)
	}()
	errorCh := make(chan error)
	defer func() {
		fmt.Println("closing errorCh")
		close(errorCh)
	}()

	go q.scanPassword(inputCh, errorCh)

	select {
	case input := <-inputCh:
		return input, nil
	case err := <-errorCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

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
