package manifest

import "testing"

func TestValidate_KindSource(t *testing.T) {
	ok := &Manifest{Packages: []Package{
		{ID: "git", Kind: "formula"},
		{ID: "figma", Kind: "cask"},
		{ID: "auto"},
		{ID: "n", Kind: "CASK", Source: "brew"},
	}}
	if err := Validate(ok); err != nil {
		t.Errorf("valid kinds should pass, got %v", err)
	}
	badKind := &Manifest{Packages: []Package{{ID: "x", Kind: "bogus"}}}
	if err := Validate(badKind); err == nil {
		t.Error("bogus kind should fail validation")
	}
	badSrc := &Manifest{Packages: []Package{{ID: "x", Source: "apt"}}}
	if err := Validate(badSrc); err == nil {
		t.Error("bogus source should fail validation")
	}
}
