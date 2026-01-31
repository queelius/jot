package entry

import (
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

func init() {
	// Configure slug generation
	slug.MaxLength = 50
	slug.Lowercase = true
}

// GenerateSlug creates a slug from a title and timestamp.
// Format: YYYYMMDD-slugified-title
// Example: "API Redesign for v2" at 2024-01-02 -> "20240102-api-redesign-for-v2"
func GenerateSlug(title string, t time.Time) string {
	datePrefix := t.Format("20060102")
	titleSlug := slug.Make(title)
	if titleSlug == "" {
		titleSlug = "untitled"
	}
	return fmt.Sprintf("%s-%s", datePrefix, titleSlug)
}

// PathForSlug returns the relative path for an entry slug.
// Example: "20240102-api-redesign" -> "entries/2024/01/20240102-api-redesign.md"
func PathForSlug(slug string) (string, error) {
	if len(slug) < 8 {
		return "", fmt.Errorf("invalid slug: too short")
	}

	// Parse date from slug prefix
	dateStr := slug[:8]
	t, err := time.Parse("20060102", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid slug date prefix: %w", err)
	}

	year := t.Format("2006")
	month := t.Format("01")

	return fmt.Sprintf("entries/%s/%s/%s.md", year, month, slug), nil
}

// SidecarPath returns the path for an entry's sidecar metadata file.
// Example: "entries/2024/01/20240102-api-redesign.md" -> "entries/2024/01/20240102-api-redesign.meta.yaml"
func SidecarPath(entryPath string) string {
	return strings.TrimSuffix(entryPath, ".md") + ".meta.yaml"
}

// AssetDir returns the path for an entry's asset directory.
// Example: "entries/2024/01/20240102-api-redesign.md" -> "entries/2024/01/20240102-api-redesign/"
func AssetDir(entryPath string) string {
	return strings.TrimSuffix(entryPath, ".md")
}
