package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseBackoffStrengths(t *testing.T) {
	got, err := parseBackoffStrengths("0, 1.5, 20")
	if err != nil {
		t.Fatalf("parseBackoffStrengths() error = %v", err)
	}
	want := []float64{0, 1.5, 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("strengths = %v, want %v", got, want)
	}
	if _, err := parseBackoffStrengths("1,-2"); err == nil {
		t.Fatal("expected negative backoff strength to be rejected")
	}
}

func TestLoadRatings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ratings.csv")
	if err := os.WriteFile(path, []byte("name,rating\nLora,5\nMira,2\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ratings, err := loadRatings(path)
	if err != nil {
		t.Fatalf("loadRatings() error = %v", err)
	}
	if len(ratings) != 2 || ratings[0].Name != "Lora" || ratings[0].Value != 5 {
		t.Fatalf("ratings = %+v, want parsed CSV rows", ratings)
	}
}
