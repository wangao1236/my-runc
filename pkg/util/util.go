package util

import (
	"os/exec"
)

func ShowProcessOne() (string, error) {
	cmd := exec.Command("ps", "-c", "1")
	output, err := cmd.Output()
	return string(output), err
}
