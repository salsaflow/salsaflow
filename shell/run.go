package shell

import (
	"bytes"
	"os/exec"
)

func Run(name string, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	return
}

func Command(name string, args ...string) (cmd *exec.Cmd, stdout, stderr *bytes.Buffer) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	cmd = exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return
}
