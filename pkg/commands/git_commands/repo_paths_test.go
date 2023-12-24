package git_commands

import (
	"testing"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

type Scenario struct {
	Name       string
	BeforeFunc func(runner *oscommands.FakeCmdObjRunner)
	Path       string
	Expected   *RepoPaths
	Err        error
}

func TestGetRepoPaths(t *testing.T) {
	scenarios := []Scenario{
		{
			Name: "typical case",
			BeforeFunc: func(runner *oscommands.FakeCmdObjRunner) {
				// setup for main worktree
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-toplevel"}, "/path/to/repo", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-dir"}, "/path/to/repo/.git", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, "/path/to/repo/.git", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-superproject-working-tree"}, "", nil)
			},
			Path: "/path/to/repo",
			Expected: &RepoPaths{
				currentPath:        "/path/to/repo",
				worktreePath:       "/path/to/repo",
				worktreeGitDirPath: "/path/to/repo/.git",
				repoPath:           "/path/to/repo",
				repoGitDirPath:     "/path/to/repo/.git",
				repoName:           "repo",
			},
			Err: nil,
		},
		{
			Name: "linked worktree",
			BeforeFunc: func(runner *oscommands.FakeCmdObjRunner) {
				// setup for linked worktree
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-toplevel"}, "/path/to/repo/worktree1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-dir"}, "/path/to/repo/.git/worktrees/worktree1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, "/path/to/repo/.git", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-superproject-working-tree"}, "", nil)
			},
			Path: "/path/to/repo/worktree1",
			Expected: &RepoPaths{
				currentPath:        "/path/to/repo/worktree1",
				worktreePath:       "/path/to/repo/worktree1",
				worktreeGitDirPath: "/path/to/repo/.git/worktrees/worktree1",
				repoPath:           "/path/to/repo",
				repoGitDirPath:     "/path/to/repo/.git",
				repoName:           "repo",
			},
			Err: nil,
		},
		{
			Name: "submodule",
			BeforeFunc: func(runner *oscommands.FakeCmdObjRunner) {
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-toplevel"}, "/path/to/repo/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-dir"}, "/path/to/repo/.git/modules/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, "/path/to/repo/.git/modules/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-superproject-working-tree"}, "/path/to/repo", nil)
			},
			Path: "/path/to/repo/submodule1",
			Expected: &RepoPaths{
				currentPath:        "/path/to/repo/submodule1",
				worktreePath:       "/path/to/repo/submodule1",
				worktreeGitDirPath: "/path/to/repo/.git/modules/submodule1",
				repoPath:           "/path/to/repo/submodule1",
				repoGitDirPath:     "/path/to/repo/.git/modules/submodule1",
				repoName:           "submodule1",
			},
			Err: nil,
		},
		{
			Name: "submodule in nested directory",
			BeforeFunc: func(runner *oscommands.FakeCmdObjRunner) {
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-toplevel"}, "/path/to/repo/my/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-dir"}, "/path/to/repo/.git/modules/my/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--git-common-dir"}, "/path/to/repo/.git/modules/my/submodule1", nil)
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-superproject-working-tree"}, "/path/to/repo", nil)
			},
			Path: "/path/to/repo/my/submodule1",
			Expected: &RepoPaths{
				currentPath:        "/path/to/repo/my/submodule1",
				worktreePath:       "/path/to/repo/my/submodule1",
				worktreeGitDirPath: "/path/to/repo/.git/modules/my/submodule1",
				repoPath:           "/path/to/repo/my/submodule1",
				repoGitDirPath:     "/path/to/repo/.git/modules/my/submodule1",
				repoName:           "submodule1",
			},
			Err: nil,
		},
		{
			Name: "git rev-parse returns an error",
			BeforeFunc: func(runner *oscommands.FakeCmdObjRunner) {
				runner.ExpectGitArgs([]string{"rev-parse", "--path-format=absolute", "--show-toplevel"}, "", errors.New("fatal: invalid gitfile format: /path/to/repo/worktree2/.git"))
			},
			Path:     "/path/to/repo/worktree2",
			Expected: nil,
			Err:      errors.New("'git rev-parse --show-toplevel' failed: fatal: invalid gitfile format: /path/to/repo/worktree2/.git"),
		},
	}

	for _, s := range scenarios {
		s := s
		t.Run(s.Name, func(t *testing.T) {
			runner := oscommands.NewFakeRunner(t)
			cmd := oscommands.NewDummyCmdObjBuilder(runner)

			// prepare the filesystem for the scenario
			s.BeforeFunc(runner)

			// run the function with the scenario path
			repoPaths, err := GetRepoPaths(cmd, s.Path)

			// check the error and the paths
			if s.Err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, s.Err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, s.Expected, repoPaths)
			}
		})
	}
}
