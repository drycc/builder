package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePreReceiveHook(t *testing.T) {
	const gitHome = "TestGitHome"
	gopath := os.Getenv("GOPATH")
	repoPath := filepath.Join(gopath, "src", "github.com", "drycc", "builder", "testdata")
	assert.Equal(t, createPreReceiveHook(gitHome, repoPath), nil)
	hookBytes, err := os.ReadFile(filepath.Join(repoPath, "hooks", "pre-receive"))
	assert.Equal(t, err, nil)
	hookStr := string(hookBytes)
	gitHomeIdx := strings.Index(hookStr, fmt.Sprintf("GIT_HOME=%s", gitHome))
	assert.False(t, gitHomeIdx == -1, "GIT_HOME was not found")
}
