package configpatch

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
)

func IntExp(t *testing.T, path string, name string, got int, exp int) {
	if got != exp {
		t.Errorf("%s: %s: Got %d, but expected %d", path, name, got, exp)
	}
}

func StrExp(t *testing.T, path string, name string, got string, exp string) {
	if got != exp {
		t.Errorf("%s: %s: %s", path, name, cmp.Diff(exp, got))
	}
}

func testPatchWrap(t *testing.T, outFile, refFile string) {
	a, _ := os.ReadFile(outFile)
	b, _ := os.ReadFile(refFile)

	if !bytes.Equal(a, b) {
		t.Fatalf("files differ:\n--- %s ---\n%s\n--- %s ---\n%s", outFile, a, refFile, b)
	}
}
