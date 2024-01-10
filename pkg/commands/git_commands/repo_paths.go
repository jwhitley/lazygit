package git_commands

import (
	ioFs "io/fs"
	"path"
	"strings"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/spf13/afero"
)

type RepoPaths struct {
	currentPath        string
	worktreePath       string
	worktreeGitDirPath string
	repoPath           string
	repoGitDirPath     string
	repoName           string
}

// Current working directory of the program. Currently, this will always
// be the same as WorktreePath(), but in future we may support running
// lazygit from inside a subdirectory of the worktree.
func (self *RepoPaths) CurrentPath() string {
	return self.currentPath
}

// Path to the current worktree. If we're in the main worktree, this will
// be the same as RepoPath()
func (self *RepoPaths) WorktreePath() string {
	return self.worktreePath
}

// Path of the worktree's git dir.
// If we're in the main worktree, this will be the .git dir under the RepoPath().
// If we're in a linked worktree, it will be the directory pointed at by the worktree's .git file
func (self *RepoPaths) WorktreeGitDirPath() string {
	return self.worktreeGitDirPath
}

// Path of the repo. If we're in a the main worktree, this will be the same as WorktreePath()
// If we're in a bare repo, it will be the parent folder of the bare repo
func (self *RepoPaths) RepoPath() string {
	return self.repoPath
}

// path of the git-dir for the repo.
// If this is a bare repo, it will be the location of the bare repo
// If this is a non-bare repo, it will be the location of the .git dir in
// the main worktree.
func (self *RepoPaths) RepoGitDirPath() string {
	return self.repoGitDirPath
}

// Name of the repo. Basename of the folder containing the repo.
func (self *RepoPaths) RepoName() string {
	return self.repoName
}

// Returns the repo paths for a typical repo
func MockRepoPaths(currentPath string) *RepoPaths {
	return &RepoPaths{
		currentPath:        currentPath,
		worktreePath:       currentPath,
		worktreeGitDirPath: path.Join(currentPath, ".git"),
		repoPath:           currentPath,
		repoGitDirPath:     path.Join(currentPath, ".git"),
		repoName:           "lazygit",
	}
}

func GetRepoPaths(
	cmd oscommands.ICmdObjBuilder,
	currentPath string,
) (*RepoPaths, error) {
	// worktreePath used to be the same as currentPath, but since we're
	// already asking git about these things, ask it for the real
	// worktree root path

	gitDirOutput, err := callGitRevParse(cmd, "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree")
	if err != nil {
		return nil, err
	}

	gitDirResults := strings.Split(utils.NormalizeLinefeeds(gitDirOutput), "\n")
	worktreePath := gitDirResults[0]
	worktreeGitDirPath := gitDirResults[1]
	repoGitDirPath := gitDirResults[2]

	// If we're in a submodule, --show-superproject-working-tree will return
	// a value, meaning gitDirResults will be length 4. In that case
	// return the worktree path as the repoPath. Otherwise we're in a
	// normal repo or a worktree so return the parent of the git common
	// dir (repoGitDirPath)
	isSubmodule := len(gitDirResults) == 4

	var repoPath string
	if isSubmodule {
		repoPath = worktreePath
	} else {
		repoPath = path.Dir(repoGitDirPath)
	}
	repoName := path.Base(repoPath)

	return &RepoPaths{
		currentPath:        currentPath,
		worktreePath:       worktreePath,
		worktreeGitDirPath: worktreeGitDirPath,
		repoPath:           repoPath,
		repoGitDirPath:     repoGitDirPath,
		repoName:           repoName,
	}, nil
}

func callGitRevParse(
	cmd oscommands.ICmdObjBuilder,
	gitRevArgs ...string,
) (string, error) {
	return callGitRevParseWithDir(cmd, "", gitRevArgs...)
}

func callGitRevParseWithDir(
	cmd oscommands.ICmdObjBuilder,
	dir string,
	gitRevArgs ...string,
) (string, error) {
	gitRevParse := NewGitCmd("rev-parse").Arg("--path-format=absolute").Arg(gitRevArgs...)
	if dir != "" {
		gitRevParse.Dir(dir)
	}

	gitCmd := cmd.New(gitRevParse.ToArgv()).DontLog()
	res, err := gitCmd.RunWithOutput()
	if err != nil {
		return "", errors.Errorf("'%s' failed: %v", gitCmd.ToString(), err)
	}
	return strings.TrimSpace(res), nil
}

// Returns the paths of linked worktrees
func linkedWortkreePaths(fs afero.Fs, repoGitDirPath string) []string {
	result := []string{}
	// For each directory in this path we're going to cat the `gitdir` file and append its contents to our result
	// That file points us to the `.git` file in the worktree.
	worktreeGitDirsPath := path.Join(repoGitDirPath, "worktrees")

	// ensure the directory exists
	_, err := fs.Stat(worktreeGitDirsPath)
	if err != nil {
		return result
	}

	_ = afero.Walk(fs, worktreeGitDirsPath, func(currPath string, info ioFs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		gitDirPath := path.Join(currPath, "gitdir")
		gitDirBytes, err := afero.ReadFile(fs, gitDirPath)
		if err != nil {
			// ignoring error
			return nil
		}
		trimmedGitDir := strings.TrimSpace(string(gitDirBytes))
		// removing the .git part
		worktreeDir := path.Dir(trimmedGitDir)
		result = append(result, worktreeDir)
		return nil
	})

	return result
}
