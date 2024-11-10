package llmcat

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func RenderGitRepo(url string, options *RenderDirectoryOptions) (string, error) {
	gitBinary, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "llmcat-clone-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "src")

	cloneCommand := exec.Command(gitBinary, "clone", url, repoDir)
	out, err := cloneCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(string(out))
	}

	return RenderDirectory(repoDir, options)
}
