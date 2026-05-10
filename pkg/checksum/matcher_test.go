package checksum

import "testing"

func TestMatcherInvalidPattern(t *testing.T) {
	invalidPatterns := []string{
		"[abc",     // unclosed character class
		"foo[",     // unclosed
		"foo[bar",  // unclosed
		"foo[]bar", // empty character class
		"foo[!]",   // invalid negation
	}

	for _, invalid := range invalidPatterns {
		t.Run(invalid, func(t *testing.T) {
			_, err := NewMatcher(
				WithAllow("valid*/**/*"),
				WithAllow(invalid),
			)

			assertErr(t, err)

			_, err = NewMatcher(
				WithBlock("valid*/**/*"),
				WithBlock(invalid),
			)

			assertErr(t, err)
		})
	}
}

func TestMatcherIsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		allow  []string
		path   string
		expect bool
	}{
		{
			name:   "no allow rules -> everything allowed",
			allow:  nil,
			path:   "foo/bar.txt",
			expect: true,
		},
		{
			name:   "matches allow pattern",
			allow:  []string{"*.txt"},
			path:   "file.txt",
			expect: true,
		},
		{
			name:   "does not match allow pattern",
			allow:  []string{"*.txt"},
			path:   "file.go",
			expect: false,
		},
		{
			name:   "multiple patterns one matches",
			allow:  []string{"*.go", "*.txt"},
			path:   "file.txt",
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Matcher{allow: tt.allow}
			got := m.IsAllowed(tt.path)

			if got != tt.expect {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestMatcherIsBlocked(t *testing.T) {
	tests := []struct {
		name   string
		block  []string
		path   string
		expect bool
	}{
		{
			name:   "no block rules",
			block:  nil,
			path:   "foo/bar.txt",
			expect: false,
		},
		{
			name:   "matches block pattern",
			block:  []string{"*.tmp"},
			path:   "file.tmp",
			expect: true,
		},
		{
			name:   "does not match block pattern",
			block:  []string{"*.tmp"},
			path:   "file.txt",
			expect: false,
		},
		{
			name:   "multiple patterns one matches",
			block:  []string{"*.log", "*.tmp"},
			path:   "file.tmp",
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Matcher{block: tt.block}
			got := m.IsBlocked(tt.path)

			if got != tt.expect {
				t.Fatalf("IsBlocked(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestMatcherMatch(t *testing.T) {
	tests := []struct {
		name   string
		allow  []string
		block  []string
		path   string
		expect bool
	}{
		{
			name:   "allowed and not blocked",
			allow:  []string{"*.txt"},
			block:  nil,
			path:   "file.txt",
			expect: true,
		},
		{
			name:   "allowed and not blocked - deep",
			allow:  []string{"**/*.txt"},
			block:  nil,
			path:   "foo/bar/file.txt",
			expect: true,
		},
		{
			name:   "allowed but blocked -> blocked wins",
			allow:  []string{"*.txt"},
			block:  []string{"file.txt"},
			path:   "file.txt",
			expect: false,
		},
		{
			name:   "not allowed",
			allow:  []string{"*.txt"},
			block:  nil,
			path:   "file.go",
			expect: false,
		},
		{
			name:   "not allowed deep",
			allow:  []string{"*.txt"},
			block:  nil,
			path:   "baz/xer/omg.docx",
			expect: false,
		},
		{
			name:   "not allowed abs deep",
			allow:  []string{"*.txt"},
			block:  nil,
			path:   "/tmp/TestFilteredWalk3090177219/001/baz/xer/omg.docx",
			expect: false,
		},
		{
			name:   "no allow rules but blocked",
			allow:  nil,
			block:  []string{"*.tmp"},
			path:   "file.tmp",
			expect: false,
		},
		{
			name:   "no allow rules and not blocked",
			allow:  nil,
			block:  []string{"*.tmp"},
			path:   "file.txt",
			expect: true,
		},
		{
			name:   "multiple allow and block patterns",
			allow:  []string{"*.txt", "*.go"},
			block:  []string{"main.go"},
			path:   "main.go",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Matcher{
				allow: tt.allow,
				block: tt.block,
			}

			got := m.Match(tt.path)

			if got != tt.expect {
				t.Fatalf("Match(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}
