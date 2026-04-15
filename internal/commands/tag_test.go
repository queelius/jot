package commands

import (
	"reflect"
	"testing"
)

func TestParseTagInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		useStdin  bool
		wantSlugs []string
		wantTags  []string
		wantErr   bool
	}{
		{"slug and tags", []string{"my-slug", "api,backend"}, false, []string{"my-slug"}, []string{"api", "backend"}, false},
		{"slug only", []string{"my-slug"}, false, []string{"my-slug"}, nil, false},
		{"no args", []string{}, false, nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := parseTagInput(tt.args, tt.useStdin)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(input.slugs, tt.wantSlugs) {
				t.Errorf("slugs = %v, want %v", input.slugs, tt.wantSlugs)
			}
			if !reflect.DeepEqual(input.tags, tt.wantTags) {
				t.Errorf("tags = %v, want %v", input.tags, tt.wantTags)
			}
		})
	}
}

func TestMutateAddTags(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		add     []string
		want    []string
	}{
		{"add to empty", nil, []string{"api", "backend"}, []string{"api", "backend"}},
		{"add new tags", []string{"existing"}, []string{"api", "backend"}, []string{"existing", "api", "backend"}},
		{"deduplicate", []string{"existing", "api"}, []string{"api", "backend"}, []string{"existing", "api", "backend"}},
		{"all duplicates", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutate := mutateAddTags(tt.add)
			got := mutate(tt.current)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutateAddTags(%v)(%v) = %v, want %v", tt.add, tt.current, got, tt.want)
			}
		})
	}
}

func TestMutateAddTags_NoAliasing(t *testing.T) {
	original := []string{"a", "b", "c"}
	backup := make([]string, len(original))
	copy(backup, original)

	mutate := mutateAddTags([]string{"d"})
	result := mutate(original)

	// Verify original slice was not modified
	if !reflect.DeepEqual(original, backup) {
		t.Errorf("mutateAddTags modified input slice: got %v, want %v", original, backup)
	}
	// Verify result has the new tag
	if len(result) != 4 || result[3] != "d" {
		t.Errorf("result = %v, want [a b c d]", result)
	}
}

func TestMutateRemoveTags(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		remove  []string
		want    []string
	}{
		{"remove existing", []string{"api", "backend", "v2"}, []string{"api", "v2"}, []string{"backend"}},
		{"remove nonexistent", []string{"api", "backend"}, []string{"missing"}, []string{"api", "backend"}},
		{"remove all", []string{"api"}, []string{"api"}, nil},
		{"remove from empty", nil, []string{"api"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutate := mutateRemoveTags(tt.remove)
			got := mutate(tt.current)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutateRemoveTags(%v)(%v) = %v, want %v", tt.remove, tt.current, got, tt.want)
			}
		})
	}
}

func TestMutateSetTags(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		set     []string
		want    []string
	}{
		{"replace tags", []string{"old1", "old2"}, []string{"new1", "new2"}, []string{"new1", "new2"}},
		{"clear tags", []string{"old1", "old2"}, nil, nil},
		{"set on empty", nil, []string{"new1"}, []string{"new1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutate := mutateSetTags(tt.set)
			got := mutate(tt.current)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutateSetTags(%v)(%v) = %v, want %v", tt.set, tt.current, got, tt.want)
			}
		})
	}
}
