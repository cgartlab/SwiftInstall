package advice

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeChat struct {
	gotUser string
	reply   string
	err     error
}

func (f *fakeChat) Chat(_ context.Context, _ string, user string) (string, error) {
	f.gotUser = user
	return f.reply, f.err
}

func TestExplain(t *testing.T) {
	f := &fakeChat{reply: "  这是一个截图工具。  "}
	out, err := Explain(context.Background(), f, "CleanShot X")
	if err != nil {
		t.Fatal(err)
	}
	if out != "这是一个截图工具。" {
		t.Errorf("reply not trimmed: %q", out)
	}
	if !strings.Contains(f.gotUser, "CleanShot X") {
		t.Errorf("prompt should include app name: %q", f.gotUser)
	}
}

func TestExplain_EmptyApp(t *testing.T) {
	if _, err := Explain(context.Background(), &fakeChat{}, "   "); err == nil {
		t.Fatal("expected error for empty app")
	}
}

func TestExplain_PropagatesError(t *testing.T) {
	f := &fakeChat{err: fmt.Errorf("boom")}
	if _, err := Explain(context.Background(), f, "X"); err == nil {
		t.Fatal("expected error propagation")
	}
}
