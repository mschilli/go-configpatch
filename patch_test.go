package configpatch

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"
)

type Action struct {
	Mode  string `yaml:"mode"`
	Regex string `yaml:"regex"`
	Text  string `yaml:"text"`
}

func TestParityCases(t *testing.T) {
	root := "test_data/cases"

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatalf("walk error: %v", err)
		}

		if !d.IsDir() {
			return nil
		}

		yamlPath := filepath.Join(path, "action.yaml")
		if _, err := os.Stat(yamlPath); err != nil {
			return nil // skip dirs without action.yaml
		}

		data, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("read %s: %v", yamlPath, err)
		}

		var a Action
		if err := yaml.Unmarshal(data, &a); err != nil {
			t.Fatalf("parse %s: %v", yamlPath, err)
		}

		inFile := filepath.Join(path, "in.txt")
		refFile := filepath.Join(path, "ref.txt")
		outFile := filepath.Join(path, "patched.txt")

		t.Logf("Patching %s", inFile)

		p := NewPatcher()
		// p.Debug = true

		err = p.Init(inFile)
		if err != nil {
			panic(err)
		}

		key := "myapp"

		h := &Hunk{
			Key:   key,
			Mode:  a.Mode,
			Regex: regexp.MustCompile(a.Regex),
			Text:  a.Text,
		}

		if err := p.Apply(h); err != nil {
			panic(err)
		}

		applyCount := 0
		if p.Patched(key) {
			applyCount = 1
		}

		if err := p.SaveAs(outFile); err != nil {
			panic(err)
		}

		testPatchWrap(t, outFile, refFile)

		count := p.Eject(key)
		IntExp(t, inFile, "eject", count, applyCount)

		if err := p.SaveAs(outFile); err != nil {
			panic(err)
		}

		testPatchWrap(t, outFile, inFile)

		err = os.Remove(outFile)
		if err != nil {
			panic(err)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func TestTraverse(t *testing.T) {
	hunks := []*Hunk{}
	texts := []string{}

	p := NewPatcher()
	path := "test_data/in.txt"
	err := p.Init(path)
	if err != nil {
		panic(err)
	}

	p.Traverse(
		func(p *Patcher, h *Hunk) {
			hunks = append(hunks, h)
		},
		func(p *Patcher, s string) {
			texts = append(texts, s)
		},
	)

	IntExp(t, path, "hunks", len(hunks), 0)
	IntExp(t, path, "texts", len(texts), 1)

	path = "test_data/out.txt"
	err = p.Init(path)
	if err != nil {
		panic(err)
	}

	p.Traverse(
		func(p *Patcher, h *Hunk) {
			hunks = append(hunks, h)
		},
		func(p *Patcher, s string) {
			texts = append(texts, s)
		},
	)

	IntExp(t, path, "hunks", len(hunks), 1)
	IntExp(t, path, "texts", len(texts), 3)
}

func TestHunk(t *testing.T) {
	h := NewHunk()
	h.Text = "foo\nbar\n"
	h.Mode = "append"
	h.Key = "myapp"

	exp := `#(Config::Patch-myapp-append)
foo
bar
#(Config::Patch-myapp-append)
`

	StrExp(t, "", "hunk", h.StringGenerate(), exp)
}

func TestReplace(t *testing.T) {
	hunks := []*Hunk{}

	p := NewPatcher()
	path := "test_data/repl.txt"
	err := p.Init(path)
	if err != nil {
		panic(err)
	}

	p.Traverse(
		func(p *Patcher, h *Hunk) {
			hunks = append(hunks, h)
		},
		func(p *Patcher, s string) {},
	)
	repl, clean, err := hunks[0].ReplaceStringExtract()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	StrExp(t, path, "repl", repl, "all:\n\techo \"foo\"\n\techo \"bar\"\n\n")
	StrExp(t, path, "clean", clean, "all:\n\techo 'all is gone!'\n")
}

func TestAppend(t *testing.T) {
	p := NewPatcher()
	path := "test_data/append/in.txt"
	err := p.Init(path)
	if err != nil {
		panic(err)
	}

	h := NewHunk()
	h.Text = "HELLO\n"
	h.Mode = "append"
	h.Key = "myapp"

	p.Apply(h)
	ref := `foo
bar
baz
#(Config::Patch-myapp-append)
HELLO
#(Config::Patch-myapp-append)
`
	StrExp(t, path, "supertest", p.Data, ref)

	count := p.Eject(h.Key)
	IntExp(t, path, "eject", count, 1)

	ref = `foo
bar
baz
`
	StrExp(t, path, h.Mode, p.Data, ref)
}

func TestMarkdown(t *testing.T) {
	p := NewPatcher()
	path := "test_data/append/in.txt"
	p.CommentStart = "<!--"
	p.CommentEnd = "-->"
	// p.Debug = true

	err := p.Init(path)
	if err != nil {
		panic(err)
	}

	h := NewHunk()
	h.Text = "HELLO\n"
	h.Mode = "append"
	h.Key = "myapp"

	p.Apply(h)
	ref := `foo
bar
baz
<!--(Config::Patch-myapp-append)-->
HELLO
<!--(Config::Patch-myapp-append)-->
`
	StrExp(t, path, "markdown-append", p.Data, ref)

	count := p.Eject(h.Key)
	IntExp(t, path, "eject", count, 1)

	ref = `foo
bar
baz
`
	StrExp(t, path, h.Mode, p.Data, ref)

	h = NewHunk()
	h.Text = "HELLO\n"
	h.Mode = "replace"
	h.Key = "myapp"
	h.Regex = regexp.MustCompile("(?m)^bar$")

	p.Apply(h)
	ref = `foo
<!--(Config::Patch-myapp-replace)-->
HELLO
<!--(Config::Patch::replace)-->
<!-- YmFyCg==-->
<!--(Config::Patch::replace)-->
<!--(Config::Patch-myapp-replace)-->
baz
`
	StrExp(t, path, "markdown-replace", p.Data, ref)

	count = p.Eject(h.Key)
	IntExp(t, path, "eject", count, 1)

	ref = `foo
bar
baz
`
	StrExp(t, path, h.Mode, p.Data, ref)
}
