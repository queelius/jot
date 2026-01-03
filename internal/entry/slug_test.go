package entry

import (
	"testing"
	"time"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title string
		time  time.Time
		want  string
	}{
		{
			title: "API Redesign for v2",
			time:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			want:  "20240102-api-redesign-for-v2",
		},
		{
			title: "Quick Thought",
			time:  time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			want:  "20241231-quick-thought",
		},
		{
			title: "",
			time:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  "20240101-untitled",
		},
		{
			title: "Title with Special Characters",
			time:  time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			want:  "20240615-title-with-special-characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := GenerateSlug(tt.title, tt.time)
			if got != tt.want {
				t.Errorf("GenerateSlug() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathForSlug(t *testing.T) {
	tests := []struct {
		slug    string
		want    string
		wantErr bool
	}{
		{
			slug: "20240102-api-redesign",
			want: "entries/2024/01/20240102-api-redesign.md",
		},
		{
			slug: "20241231-test",
			want: "entries/2024/12/20241231-test.md",
		},
		{
			slug:    "short",
			wantErr: true,
		},
		{
			slug:    "notadate-test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got, err := PathForSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("PathForSlug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PathForSlug() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSidecarPath(t *testing.T) {
	got := SidecarPath("entries/2024/01/20240102-test.md")
	want := "entries/2024/01/20240102-test.meta.yaml"
	if got != want {
		t.Errorf("SidecarPath() = %q, want %q", got, want)
	}
}

func TestAssetDir(t *testing.T) {
	got := AssetDir("entries/2024/01/20240102-test.md")
	want := "entries/2024/01/20240102-test"
	if got != want {
		t.Errorf("AssetDir() = %q, want %q", got, want)
	}
}
