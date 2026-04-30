package configpatch

import (
	"strings"
	"testing"
)

func TestHunkMarker(t *testing.T) {
	h := &Hunk{
		Key:              "mykey",
		Mode:             "replace",
		CommentStart: "#",
	}

	expected := "#(Config::Patch-mykey-replace)\n"

	if h.PatchMarker() != expected {
		t.Fatalf("marker mismatch:\n got: %q\nwant: %q", h.PatchMarker(), expected)
	}
}

func TestHunkGenerateBasic(t *testing.T) {
	h := &Hunk{
		Key:              "k",
		Mode:             "append",
		Text:             "HELLO",
		CommentStart: "#",
	}

	out := h.StringGenerate()

	expected :=
		"#(Config::Patch-k-append)\n" +
			"HELLO\n" +
			"#(Config::Patch-k-append)\n"

	if out != expected {
		t.Fatalf("generate mismatch:\n%s", out)
	}
}

func TestHunkGenerateAddsNewline(t *testing.T) {
	h := &Hunk{
		Key:              "k",
		Mode:             "append",
		Text:             "HELLO", // no newline
		CommentStart: "#",
	}

	out := h.StringGenerate()

	if !strings.Contains(out, "HELLO\n") {
		t.Fatalf("missing newline in generated output:\n%s", out)
	}
}

func TestFreezeProducesCommentedBase64(t *testing.T) {
	h := &Hunk{
		CommentStart: "#",
	}

	input := "bar\n"
	frozen := h.Freeze(input)

	// base64("bar\n") = YmFyCg==
	expected := "YmFyCg=="

	if frozen != expected {
		t.Fatalf("freeze mismatch:\n got: %q\nwant: %q", frozen, expected)
	}
}

func TestThawRestoresOriginal(t *testing.T) {
	h := &Hunk{
		CommentStart: "#",
	}

	original := "bar\n"
	frozen := h.Freeze(original)
	thawed, err := h.Thaw(frozen)
	if err != nil {
		panic(err)
	}

	if thawed != original {
		t.Fatalf("thaw mismatch:\n got: %q\nwant: %q", thawed, original)
	}
}

func TestFreezeThawRoundTrip(t *testing.T) {
	h := &Hunk{
		CommentStart: "#",
	}

	input := "foo\nbar\nbaz\n"

	frozen := h.Freeze(input)
	out, err := h.Thaw(frozen)
	if err != nil {
		panic(err)
	}

	if out != input {
		t.Fatalf("roundtrip failed:\n%s", out)
	}
}

func TestFreezeEmptyString(t *testing.T) {
	h := &Hunk{
		CommentStart: "#",
	}

	frozen := h.Freeze("")
	if frozen != "" {
		t.Fatalf("unexpected freeze of empty string: %q", frozen)
	}
}

func TestThawEmpty(t *testing.T) {
	h := &Hunk{
		CommentStart: "#",
	}

	out, err := h.Thaw("")
	if err != nil {
		panic(err)
	}

	if out != "" {
		t.Fatalf("unexpected thaw result: %q", out)
	}
}
