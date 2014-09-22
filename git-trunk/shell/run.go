package shell

import (
	"bytes"
	"os/exec"
)

func Run(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()

	return
}
