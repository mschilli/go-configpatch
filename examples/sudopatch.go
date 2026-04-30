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
