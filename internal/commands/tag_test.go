package commands

import (
	"reflect"
	"testing"
)

func TestGetTagsArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		useStdin bool
		want     string
	}{
		{"no stdin, slug and tags", []string{"my-slug", "api,backend"}, false, "api,backend"},
		{"no stdin, slug only", []string{"my-slug"}, false, ""},
		{"stdin, tags as first arg", []string{"api,backend"}, true, "api,backend"},
		{"stdin, no args", []string{}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTagsArg(tt.args, tt.useStdin)
			if got != tt.want {
				t.Errorf("getTagsArg(%v, %v) = %q, want %q", tt.args, tt.useStdin, got, tt.want)
			}
		})
	}
}

func TestGetSlugInputs_Args(t *testing.T) {
	t.Run("from positional arg", func(t *testing.T) {
		slugs, err := getSlugInputs([]string{"my-slug", "tags"}, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(slugs) != 1 || slugs[0] != "my-slug" {
			t.Errorf("got %v, want [my-slug]", slugs)
		}
	})

	t.Run("no args, no stdin", func(t *testing.T) {
		_, err := getSlugInputs([]string{}, false)
		if err == nil {
			t.Error("expected error for no args")
		}
	})
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
