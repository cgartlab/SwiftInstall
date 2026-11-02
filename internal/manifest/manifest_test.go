package manifest

import (
	"reflect"
	"testing"
)

func TestDedupe_KeepsFirstOccurrenceInOrder(t *testing.T) {
	in := []Package{
		{ID: "a", Category: "first"},
		{ID: "b", Category: "b-only"},
		{ID: "a", Category: "second"},
		{ID: "c", Category: "c-only"},
		{ID: "b", Category: "b-again"},
	}
	got := Dedupe(in)
	want := []Package{
		{ID: "a", Category: "first"},
		{ID: "b", Category: "b-only"},
		{ID: "c", Category: "c-only"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Dedupe = %#v, want %#v", got, want)
	}
}

func TestDedupe_NoDuplicates(t *testing.T) {
	in := []Package{{ID: "x"}, {ID: "y"}}
	got := Dedupe(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("Dedupe = %#v, want %#v", got, in)
	}
}

func TestDedupe_Empty(t *testing.T) {
	if got := Dedupe(nil); len(got) != 0 {
		t.Fatalf("Dedupe(nil) len = %d, want 0", len(got))
	}
	if got := Dedupe([]Package{}); len(got) != 0 {
		t.Fatalf("Dedupe(empty) len = %d, want 0", len(got))
	}
}

func TestValidate(t *testing.T) {
	valid := &Manifest{Packages: []Package{{ID: "Git.Git"}}}

	tests := []struct {
		name    string
		m       *Manifest
		wantErr bool
	}{
		{name: "nil manifest", m: nil, wantErr: true},
		{name: "no packages", m: &Manifest{}, wantErr: true},
		{name: "empty ID", m: &Manifest{Packages: []Package{{ID: "  "}}}, wantErr: true},
		{name: "negative retry_count", m: &Manifest{
			Packages: []Package{{ID: "a"}},
			Settings: Settings{RetryCount: -1},
		}, wantErr: true},
		{name: "negative retry_delay", m: &Manifest{
			Packages: []Package{{ID: "a"}},
			Settings: Settings{RetryDelay: -5},
		}, wantErr: true},
		{name: "valid", m: valid, wantErr: false},
		{name: "valid with settings", m: &Manifest{
			Packages: []Package{{ID: "a"}, {ID: "b"}},
			Settings: Settings{SkipExisting: true, RetryCount: 3, RetryDelay: 2, Proxy: "http://127.0.0.1:7890"},
		}, wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.m)
			if tc.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
