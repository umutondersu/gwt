package core

import (
	"reflect"
	"testing"
)

func TestParseSessionsUnderPath(t *testing.T) {
	out := "sess-a /repo/worktree/foo\n" +
		"sess-a /repo/worktree/foo/app\n" +
		"sess-b /repo/worktree/foo/env\n" +
		"sess-c /repo/worktree/fooish\n" +
		"sess-d /repo/main\n" +
		"sess-e /repo/worktree/bar\n"

	cases := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "exact and subdirectories only",
			path: "/repo/worktree/foo",
			want: []string{"sess-a", "sess-b"},
		},
		{
			name: "subdirectory root picks the dir itself",
			path: "/repo/worktree/foo/app",
			want: []string{"sess-a"},
		},
		{
			name: "sibling prefix is not matched",
			path: "/repo/worktree/foo/en",
			want: nil,
		},
		{
			name: "no matches",
			path: "/repo/nowhere",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSessionsUnderPath(out, tc.path)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseSessionsUnderPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestParseSessionsUnderPathEmpty(t *testing.T) {
	if got := parseSessionsUnderPath("", "/repo/worktree/foo"); got != nil {
		t.Errorf("expected nil for empty output, got %v", got)
	}
	if got := parseSessionsUnderPath("  \n", "/repo/worktree/foo"); got != nil {
		t.Errorf("expected nil for whitespace output, got %v", got)
	}
}

func TestIsUnderPath(t *testing.T) {
	cases := []struct {
		cwd  string
		dir  string
		want bool
	}{
		{"/repo/worktree/foo", "/repo/worktree/foo", true},
		{"/repo/worktree/foo/app", "/repo/worktree/foo", true},
		{"/repo/worktree/fooish", "/repo/worktree/foo", false},
		{"/repo/worktree/f", "/repo/worktree/foo", false},
		{"/repo/worktree/foobar/baz", "/repo/worktree/foo", false},
		{"", "/repo/worktree/foo", false},
		{"/repo/worktree/foo", "/repo/worktree/fo", false},
		{"/repo/worktree/foo", "/repo/worktree/foo/", true},
	}

	for _, tc := range cases {
		if got := IsUnderPath(tc.cwd, tc.dir); got != tc.want {
			t.Errorf("IsUnderPath(%q, %q) = %v, want %v", tc.cwd, tc.dir, got, tc.want)
		}
	}
}
