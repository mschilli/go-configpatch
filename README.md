# configpatch

This Go package patches text files (typically configuration files) with hunks
of text that can later be easily removed or replaced. It is a port of the
Config::Patch Perl module on CPAN (see appendix for differences).

## Example: Append a line in /etc/sudoers

Let's say you have a sudoers file that looks like this:

```
# Sample /etc/sudoers file.
Defaults        env_reset
```

Now with configpatch, you can add a user in a reversible way:

<!--(Config::Patch-example1-replace)-->
go ```
package main

import (
	cnfp "github.com/mschilli/go-configpatch"
)

func main() {
	patcher := cnfp.NewPatcher()
	patcher.Init("sudoers")

	hunk := cnfp.NewHunk()
	hunk.Key = "myapp"
	hunk.Mode = "append"
	hunk.Text = "joeschmoe ALL= NOPASSWD:/etc/rc.d/init.d/myapp\n"

	patcher.Apply(hunk)
	patcher.Save()

	// Later: restore original file
	// patcher.Eject("myapp")
	// patcher.Save()
}
```
<!--(Config::Patch::replace)-->
<!-- RVhBTVBMRTEK-->
<!--(Config::Patch::replace)-->
<!--(Config::Patch-example1-replace)-->

which adds a patch under the key "myapp" and turns the file into

```
# Sample /etc/sudoers file.
Defaults        env_reset
#(Config::Patch-myapp-append)
joeschmoe ALL= NOPASSWD:/etc/rc.d/init.d/myapp
#(Config::Patch-myapp-append)
```

and hence grants joeschmoe certain permissions. Note that *configpatch* marks the patched
section, using configurable comment characters to not interfere with the syntax
of the configuration format in use. This way, *configpatch* will be able to locate the patch
later on, and you can easily remove the patch with the Eject() function in the code above,
and restore the old version with Save().

(Ironically, this README.md was patched with configpatch at release time, reading the
code for the example above, using "replace" mode with "markdown" comments.)

---

## Description

What's the use? `configpatch` allows you to modify configuration files in a way
that can be safely reversed later, to remove a patch or update it with new
data.

Instead of just applying patches (like patch), it inserts **marker blocks** around
changes. These markers allow the package to:

* detect previously applied patches
* update them
* remove them cleanly

Note that a "patch" here is not a diff in configpatch's universe, but a simple
hunk of text, to be applied line by line.

---

## Example

In general, a patch gets applied like this:

Original file:

```
...
previous content
...
```

After applying a patch:

```
...
previous content
...
#(Config::Patch-myapp-append)
joeschmoe ALL= NOPASSWD:/etc/rc.d/init.d/myapp
#(Config::Patch-myapp-append)
```

These markers:

* are commented out (using # by default, but this is configurable)
* Name the patch ("myapp"), so that multiple patches under different names can
  be applied
* ignored by the target application consuming the config file
* used internally by configpatch for detection/rollback later

---

## API Usage

### Create a Patcher

```go
patcher := patch.NewPatcher("/etc/sudoers")
```

### Apply a Patch

```go
patcher.Apply(hunk)
patcher.Save()
```

### Remove patch

```go
patcher.Eject("myapp")
patcher.Save()
```

### Save to different file

```go
patcher.SaveAs("newfile.dat")
```

### Inspect current data

```go
data := patcher.Data
fmt.Println(data)
```

---

## Patch Modes

---

### `prepend`

Insert at beginning. The hunk of text in the patch is added to the file before
the 1st line.

---

### `replace`

Replace matching lines in the file and store their original content Base64 encoded,
so it can be resurrected later.

```go
hunk := patch.NewHunk(patch.Hunk{
    Key:   "myapp",
    Mode:  "replace",
    Regex: regexp.MustCompile(`(?ms)^all:.*?\n\n`),
    Text:  "all:\n\techo 'all is gone!'\n",
})
```

Result:

```
#(Config::Patch-myapp-replace)
all:
    echo 'all is gone!'
#(Config::Patch::replace)
# <base64 original>
#(Config::Patch::replace)
#(Config::Patch-myapp-replace)
```

Rollback restores original content.

---

### `insert-after`

```go
hunk := patch.NewHunk(patch.Hunk{
    Key:   "myapp",
    Mode:  "insert-after",
    Regex: regexp.MustCompile(`(?m)^\[section\]`),
    Text:  "foo=bar\n",
})
```

---

### `insert-before`

```go
hunk := patch.NewHunk(patch.Hunk{
    Key:   "myapp",
    Mode:  "insert-before",
    Regex: regexp.MustCompile(`(?m)^\[section\]`),
    Text:  "[newsection]\nfoo=bar\n\n",
})
```

---

### `update`

Update an existing patch in place:

```go
hunk := patch.NewHunk(patch.Hunk{
    Key:  "myapp",
    Mode: "update",
    Text: "foo=woot\n",
})
```

---

## Regex

All match locations are determined via Go regex:

```go
regexp.MustCompile(`(?m)^pattern`)
```

---

## Newline Handling

* Every line must end with `\n`
* Missing newline → automatically added
* Applies to both file content and patch text

---

## Using different formats for comments

Default: `#`

Custom:

```go
p := patch.NewPatcher
p.CommentStart = "<!--"
p.CommentEnd = "-->"
p.Init("myfile.dat")
```

---

## Notes

* Only one patch per key, dupes are rejected
* Patches are **line-based**
* No external metadata required, all data self-contained in the patched file

## Porting Notes

This is a Go port of the original Config::Patch CPAN module
(https://metacpan.org/pod/Config::Patch).
Its implementation is meant to provide complete functional parity, except
for the following:

* Config::Patch won't add a trailing "\n" to the hunk text in append or prepend mode.
  This is a bug in the original, fixed in the Go port.

* Custom comment formats are new to the Go port, they might find their way to the
  original as time allows (cough, cough).

## Author

Mike Schilli, m@perlmeister.com 2026

## License

Released under the [Apache 2.0](LICENSE)
