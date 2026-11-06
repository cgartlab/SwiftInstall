package toollist

import "testing"

func TestLoadBuiltin(t *testing.T) {
	c, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin: %v", err)
	}
	if c.ID == "" {
		t.Fatal("catalog has empty id")
	}
	if len(c.Items) == 0 {
		t.Fatal("catalog has no items")
	}
	if len(c.Equivalences) == 0 {
		t.Fatal("catalog has no equivalences")
	}
	// every item must record why it qualifies
	for _, it := range c.Items {
		if it.Name == "" || it.Why == "" {
			t.Errorf("item missing name/why: %+v", it)
		}
	}
}
