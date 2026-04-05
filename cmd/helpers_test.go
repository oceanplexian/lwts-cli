package cmd

import (
	"testing"
)

func TestMapTag(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature", "blue"},
		{"feat", "blue"},
		{"Feature", "blue"},
		{"FEAT", "blue"},
		{"fix", "green"},
		{"bugfix", "green"},
		{"infra", "orange"},
		{"infrastructure", "orange"},
		{"ops", "orange"},
		{"bug", "red"},
		{"defect", "red"},
		// passthrough for literal colors and unknown values
		{"blue", "blue"},
		{"green", "green"},
		{"orange", "orange"},
		{"red", "red"},
		{"purple", "purple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapTag(tt.input)
			if got != tt.want {
				t.Errorf("MapTag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapPriority(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// highest
		{"critical", "highest"},
		{"urgent", "highest"},
		{"p0", "highest"},
		{"Critical", "highest"},
		// high
		{"high", "high"},
		{"important", "high"},
		{"p1", "high"},
		// medium
		{"medium", "medium"},
		{"normal", "medium"},
		{"p2", "medium"},
		// low
		{"low", "low"},
		{"minor", "low"},
		{"p3", "low"},
		// lowest
		{"lowest", "lowest"},
		{"trivial", "lowest"},
		{"p4", "lowest"},
		// passthrough
		{"highest", "highest"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapPriority(tt.input)
			if got != tt.want {
				t.Errorf("MapPriority(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string
	}{
		{
			name: "empty args",
			args: []string{},
			want: map[string]string{},
		},
		{
			name: "single flag",
			args: []string{"--tag=blue"},
			want: map[string]string{"tag": "blue"},
		},
		{
			name: "multiple flags",
			args: []string{"--tag=blue", "--priority=high", "--points=5"},
			want: map[string]string{"tag": "blue", "priority": "high", "points": "5"},
		},
		{
			name: "ignores non-flag args",
			args: []string{"some-title", "--tag=red", "extra"},
			want: map[string]string{"tag": "red"},
		},
		{
			name: "flag with empty value",
			args: []string{"--tag="},
			want: map[string]string{"tag": ""},
		},
		{
			name: "flag with equals in value",
			args: []string{"--desc=a=b=c"},
			want: map[string]string{"desc": "a=b=c"},
		},
		{
			name: "single dash ignored",
			args: []string{"-tag=blue"},
			want: map[string]string{},
		},
		{
			name: "flag without equals ignored",
			args: []string{"--verbose"},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFlags(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("ParseFlags(%v) returned %d entries, want %d", tt.args, len(got), len(tt.want))
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("ParseFlags(%v) missing key %q", tt.args, k)
				} else if gotV != wantV {
					t.Errorf("ParseFlags(%v)[%q] = %q, want %q", tt.args, k, gotV, wantV)
				}
			}
		})
	}
}

func TestFlagOr(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		key   string
		def   string
		want  string
	}{
		{
			name:  "key present",
			flags: map[string]string{"column": "done"},
			key:   "column",
			def:   "todo",
			want:  "done",
		},
		{
			name:  "key missing uses default",
			flags: map[string]string{},
			key:   "column",
			def:   "todo",
			want:  "todo",
		},
		{
			name:  "empty value uses default",
			flags: map[string]string{"column": ""},
			key:   "column",
			def:   "todo",
			want:  "todo",
		},
		{
			name:  "nil map uses default",
			flags: map[string]string{},
			key:   "anything",
			def:   "fallback",
			want:  "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlagOr(tt.flags, tt.key, tt.def)
			if got != tt.want {
				t.Errorf("FlagOr(%v, %q, %q) = %q, want %q", tt.flags, tt.key, tt.def, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "within limit",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "at limit",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "over limit",
			s:    "hello world, this is a long string",
			max:  10,
			want: "hello w...",
		},
		{
			name: "empty string",
			s:    "",
			max:  10,
			want: "",
		},
		{
			name: "exactly one over",
			s:    "abcdef",
			max:  5,
			want: "ab...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
