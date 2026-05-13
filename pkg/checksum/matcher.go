package checksum

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

type Matcher struct {
	allow []string
	block []string
}

type MatcherOption func(*Matcher) error

func WithAllow(pattern string) MatcherOption {
	// NOTE: convert to native separators so we also support '/' on windows
	pattern = filepath.FromSlash(pattern)
	return func(m *Matcher) error {
		if !doublestar.ValidatePathPattern(pattern) {
			return fmt.Errorf("invalid allow pattern: %q", pattern)
		}
		m.allow = append(m.allow, pattern)
		return nil
	}
}

func WithBlock(pattern string) MatcherOption {
	// NOTE: convert to native separators so we also support '/' on windows
	pattern = filepath.FromSlash(pattern)
	return func(m *Matcher) error {
		if !doublestar.ValidatePathPattern(pattern) {
			return fmt.Errorf("invalid block pattern: %q", pattern)
		}
		m.block = append(m.block, pattern)
		return nil
	}
}

func NewMatcher(opts ...MatcherOption) (Matcher, error) {
	var m Matcher
	for _, opt := range opts {
		if err := opt(&m); err != nil {
			return Matcher{}, err
		}
	}
	return m, nil
}

func (m *Matcher) IsAllowed(path string) bool {
	if len(m.allow) == 0 {
		return true
	}

	for _, patt := range m.allow {
		match := doublestar.PathMatchUnvalidated(patt, path)
		if match {
			return true
		}
	}

	return false
}

func (m *Matcher) IsBlocked(path string) bool {
	for _, patt := range m.block {
		match := doublestar.PathMatchUnvalidated(patt, path)
		if match {
			return true
		}
	}

	return false
}

func (m *Matcher) Match(path string) bool {
	if m.IsAllowed(path) && !m.IsBlocked(path) {
		return true
	}

	return false
}
