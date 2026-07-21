// Package finder provides utilities for finding and selecting items using fuzzy search
// and ID-based lookups. It is primarily used for interactive selection of Azure
// resources like tenants and subscriptions.
package finder

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	fzf "github.com/junegunn/fzf/src"
)

// ErrAbort is returned when the user aborts the picker without choosing.
var ErrAbort = errors.New("abort")

// configured holds extra fzf options from the user's config (picker.options);
// previewEnabled gates the details preview pane (picker.preview, default off).
var (
	configured     []string
	previewEnabled bool
)

// Configure sets additional fzf options and the preview toggle for all
// subsequent Fuzzy calls.
func Configure(options []string, preview bool) {
	configured = options
	previewEnabled = preview
}

// IDGetter is an interface that both Tenant and Subscription implement
type IDGetter interface {
	GetID() uuid.UUID
}

// Fuzzy provides interactive fuzzy finding using fzf as an embedded library.
// The picker renders inline (adaptive height, at most 40%% of the screen) and
// honors FZF_DEFAULT_OPTS plus the user's picker.options from the config.
func Fuzzy[T any](items []T, displayFunc func(T) string) (*T, error) {
	return run(items, displayFunc, nil)
}

// FuzzyPreview is Fuzzy with a preview pane rendering previewFunc's text for
// the highlighted item. Users can restyle or disable it via picker.options
// (--preview-window, or --preview=” to turn it off).
func FuzzyPreview[T any](items []T, displayFunc, previewFunc func(T) string) (*T, error) {
	return run(items, displayFunc, previewFunc)
}

// escapePreview makes multi-line preview text safe to embed in a
// tab-delimited entry line; printf '%b' in the preview command re-expands it.
func escapePreview(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "    ")
	return strings.ReplaceAll(s, "\n", "\\n")
}

func run[T any](items []T, displayFunc, previewFunc func(T) string) (*T, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select from")
	}
	if !previewEnabled {
		previewFunc = nil
	}

	// Adaptive height shrinks to the item count, which would clip the
	// preview pane — use a fixed fraction when a preview is shown.
	args := []string{"--height=~40%", "--layout=reverse"}
	if previewFunc != nil {
		args = []string{"--height=40%", "--layout=reverse",
			"--preview", "printf '%b' {3}", "--preview-window", "right,55%"}
	}
	args = append(args, configured...)
	// Non-negotiable tail: entries are fed as "<index>\t<display>[\t<preview>]"
	// so the pick maps back to an item even when display strings collide.
	args = append(args, "--delimiter=\t", "--with-nth=2", "--no-multi")

	options, err := fzf.ParseOptions(true, args)
	if err != nil {
		return nil, fmt.Errorf("invalid picker options: %w", err)
	}

	input := make(chan string)
	go func() {
		defer close(input)
		for i := range items {
			line := fmt.Sprintf("%d\t%s", i, displayFunc(items[i]))
			if previewFunc != nil {
				line += "\t" + escapePreview(previewFunc(items[i]))
			}
			input <- line
		}
	}()
	output := make(chan string, len(items))
	options.Input = input
	options.Output = output

	code, err := fzf.Run(options)
	if err != nil {
		return nil, err
	}
	switch code {
	case fzf.ExitOk:
	case fzf.ExitInterrupt, fzf.ExitNoMatch:
		return nil, ErrAbort
	default:
		return nil, fmt.Errorf("picker exited with code %d", code)
	}

	select {
	case line := <-output:
		idxStr, _, _ := strings.Cut(line, "\t")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(items) {
			return nil, fmt.Errorf("unexpected picker output %q", line)
		}
		return &items[idx], nil
	default:
		return nil, ErrAbort
	}
}

// ByID finds an item by its UUID in a slice of items that implement IDGetter
func ByID[T IDGetter](items []T, id uuid.UUID) (*T, error) {
	for _, item := range items {
		if item.GetID() == id {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("item not found")
}
