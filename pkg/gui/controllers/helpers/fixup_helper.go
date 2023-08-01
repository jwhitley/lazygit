package helpers

import (
	"regexp"
	"strings"
	"sync"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type FixupHelper struct {
	c *HelperCommon
}

func NewFixupHelper(
	c *HelperCommon,
) *FixupHelper {
	return &FixupHelper{
		c: c,
	}
}

type deletedLineInfo struct {
	filename     string
	startLineIdx int
	numLines     int
}

func (self *FixupHelper) HandleFindBaseCommitForFixupPress() error {
	diff, useIndex, err := self.getDiff()
	if err != nil {
		return err
	}
	if diff == "" {
		return self.c.ErrorMsg("No changes to commit")
	}

	deletedLineInfos := self.parseDiff(diff)
	if len(deletedLineInfos) == 0 {
		return self.c.ErrorMsg("No deleted lines in diff")
	}

	shas := self.blameDeletedLines(deletedLineInfos)

	if len(shas) == 0 {
		// This should never happen
		return self.c.ErrorMsg("No base commits found")
	}
	if len(shas) > 1 {
		subjects, err := self.c.Git().Commit.GetShasAndCommitMessagesFirstLine(shas)
		if err != nil {
			return err
		}
		return self.c.ErrorMsg("Multiple base commits found. (" +
			lo.Ternary(useIndex, "Try staging fewer changes", "Try staging some of the changes") +
			")\n\n" + subjects)
	}

	commit, index, ok := lo.FindIndexOf(self.c.Model().Commits, func(commit *models.Commit) bool {
		return commit.Sha == shas[0]
	})
	if !ok {
		if self.c.Model().Commits[len(self.c.Model().Commits)-1].Status == models.StatusMerged {
			// If the commit is not found, it's most likely because it's already
			// merged, and more than 300 commits away. Check if the last known
			// commit is already merged; if so, show the "already merged" error.
			return self.c.ErrorMsg("The base commit for this change is already on master")
		}
		// If we get here, the current branch must have more then 300 commits. Unlikely...
		return self.c.ErrorMsg("Base commit is not in current view")
	}
	if commit.Status == models.StatusMerged {
		return self.c.ErrorMsg("The base commit for this change is already on master")
	}

	if !useIndex {
		if err := self.c.Git().WorkingTree.StageAll(); err != nil {
			return err
		}
		_ = self.c.Refresh(types.RefreshOptions{Mode: types.SYNC, Scope: []types.RefreshableView{types.FILES}})
	}

	self.c.Contexts().LocalCommits.SetSelectedLineIdx(index)
	return self.c.PushContext(self.c.Contexts().LocalCommits)
}

func (self *FixupHelper) getDiff() (string, bool, error) {
	args := []string{"-U0", "--ignore-submodules=all", "HEAD", "--"}

	// Try staged changes first
	useIndex := true
	diff, err := self.c.Git().Diff.DiffIndexCmdObj(append([]string{"--cached"}, args...)...).RunWithOutput()

	if err == nil && diff == "" {
		useIndex = false
		// If there are no staged changes, try unstaged changes
		diff, err = self.c.Git().Diff.DiffIndexCmdObj(args...).RunWithOutput()
	}

	return diff, useIndex, err
}

func (self *FixupHelper) parseDiff(diff string) []*deletedLineInfo {
	lines := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")

	deletedLineInfos := []*deletedLineInfo{}

	hunkHeaderRegexp := regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+\d+(?:,\d+)? @@`)

	var filename string
	var currentLineInfo *deletedLineInfo
	finishHunk := func() {
		if currentLineInfo != nil && currentLineInfo.numLines > 0 {
			deletedLineInfos = append(deletedLineInfos, currentLineInfo)
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			finishHunk()
			currentLineInfo = nil
		} else if strings.HasPrefix(line, "--- ") {
			// For some reason, the line ends with a tab character if the file
			// name contains spaces
			filename = strings.TrimRight(line[6:], "\t")
		} else if strings.HasPrefix(line, "@@ ") {
			finishHunk()
			match := hunkHeaderRegexp.FindStringSubmatch(line)
			startIdx := utils.MustConvertToInt(match[1])
			currentLineInfo = &deletedLineInfo{filename, startIdx, 0}
		} else if currentLineInfo != nil && line[0] == '-' {
			currentLineInfo.numLines++
		}
	}
	finishHunk()

	return deletedLineInfos
}

func (self *FixupHelper) blameDeletedLines(deletedLineInfos []*deletedLineInfo) []string {
	var wg sync.WaitGroup
	shaChan := make(chan string)

	for _, info := range deletedLineInfos {
		wg.Add(1)
		go func(info *deletedLineInfo) {
			defer wg.Done()

			blameOutput, err := self.c.Git().Blame.BlameLineRange(info.filename, "HEAD", info.startLineIdx, info.numLines)
			if err != nil {
				self.c.Log.Errorf("Error blaming file '%s': %v", info.filename, err)
				return
			}
			blameLines := strings.Split(strings.TrimSuffix(blameOutput, "\n"), "\n")
			for _, line := range blameLines {
				shaChan <- strings.Split(line, " ")[0]
			}
		}(info)
	}

	go func() {
		wg.Wait()
		close(shaChan)
	}()

	result := set.New[string]()
	for sha := range shaChan {
		result.Add(sha)
	}

	return result.ToSlice()
}
