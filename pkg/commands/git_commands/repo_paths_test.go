package git_commands

import (
	"strings"
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
				expectedOutput := []string{
					// --show-toplevel
					"/path/to/repo",
					// --git-dir
					"/path/to/repo/.git",
					// --git-common-dir
					"/path/to/repo/.git",
					// --show-superproject-working-tree
				}
				runner.ExpectGitArgs(
					[]string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree"},
					strings.Join(expectedOutput, "\n"),
					nil)
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
				expectedOutput := []string{
					// --show-toplevel
					"/path/to/repo/worktree1",
					// --git-dir
					"/path/to/repo/.git/worktrees/worktree1",
					// --git-common-dir
					"/path/to/repo/.git",
					// --show-superproject-working-tree
				}
				runner.ExpectGitArgs(
					[]string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree"},
					strings.Join(expectedOutput, "\n"),
					nil)
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
				expectedOutput := []string{
					// --show-toplevel
					"/path/to/repo/submodule1",
					// --git-dir
					"/path/to/repo/.git/modules/submodule1",
					// --git-common-dir
					"/path/to/repo/.git/modules/submodule1",
					// --show-superproject-working-tree
					"/path/to/repo",
				}
				runner.ExpectGitArgs(
					[]string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree"},
					strings.Join(expectedOutput, "\n"),
					nil)
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
				expectedOutput := []string{
					// --show-toplevel
					"/path/to/repo/my/submodule1",
					// --git-dir
					"/path/to/repo/.git/modules/my/submodule1",
					// --git-common-dir
					"/path/to/repo/.git/modules/my/submodule1",
					// --show-superproject-working-tree
					"/path/to/repo",
				}
				runner.ExpectGitArgs(
					[]string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree"},
					strings.Join(expectedOutput, "\n"),
					nil)
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
				runner.ExpectGitArgs(
					[]string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir", "--show-superproject-working-tree"},
					"",
					errors.New("fatal: invalid gitfile format: /path/to/repo/worktree2/.git"))
			},
			Path:     "/path/to/repo/worktree2",
			Expected: nil,
			Err:      errors.New("'git rev-parse --path-format=absolute --show-toplevel --git-dir --git-common-dir --show-superproject-working-tree' failed: fatal: invalid gitfile format: /path/to/repo/worktree2/.git"),
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
