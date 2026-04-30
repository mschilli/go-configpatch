package main

import (
	"flag"
	"fmt"
	"github.com/mschilli/go-configpatch"
	"io"
	"os"
	"path"
	"regexp"
)

const Version = "0.01"

func main() {
	append := flag.Bool("append", false, "append")
	eject := flag.Bool("eject", false, "eject a patch")
	replace := flag.String("replace", "", "replace by regex")
	key := flag.String("key", "", "patch key")
	comments := flag.String("comments", "", "# or markdown")
	debug := flag.Bool("debug", false, "enable verbose logging")
	version := flag.Bool("version", false, "print version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: echo lines | %s [-append|-replace|-eject] -key=key file\n",
			path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Printf("%s\n", Version)
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "No file to patch provided.\n\n")
		flag.Usage()
		return
	}

	if len(*key) == 0 {
		fmt.Fprintf(os.Stderr, "No key provided.\n\n")
		flag.Usage()
		return
	}

	path := flag.Arg(0)

	patcher := configpatch.NewPatcher()
	if *debug {
		patcher.Debug = true
	}

	if len(*comments) != 0 {
		switch *comments {
		case "markdown":
			patcher.CommentStartChar = "<!--"
			patcher.CommentEndChar = "-->"
		default:
			panic(fmt.Errorf("Unknown comment format: %s", *comments))
		}
	}

	err := patcher.Init(path)
	if err != nil {
		panic(err)
	}

	h := &configpatch.Hunk{
		Key: *key,
	}

	if *append {
		h.Mode = "append"
		h.Text = slurpStdin()
		if err := patcher.Apply(h); err != nil {
			panic(err)
		}
	} else if len(*replace) != 0 {
		h.Mode = "replace"
		h.Regex = regexp.MustCompile(*replace)
		h.Text = slurpStdin()
		if err := patcher.Apply(h); err != nil {
			panic(err)
		}
		fmt.Printf("Patch applied (%s)\n", h.Mode)
	} else if *eject {
		count := patcher.Eject(h.Key)
		fmt.Printf("%d patches ejected\n", count)
	} else {
		fmt.Printf("Unknown action\n")
		flag.Usage()
	}

	err = patcher.Save()
	if err != nil {
		panic(err)
	}
}

func slurpStdin() string {
	s, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	return string(s)
}
