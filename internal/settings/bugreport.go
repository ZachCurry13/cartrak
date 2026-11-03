package settings

import (
	"fmt"
	"regexp"
	"strings"
)

// The private GitHub repository a bug report is sent to, and the token that may write to it. The token is stored on the server only.
const (
	ReportRepoKey  = "CARTRAK_REPORT_REPO"
	ReportTokenKey = "CARTRAK_REPORT_TOKEN"
)

var reportRepoRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)

func init() {
	Defs = append(Defs,
		Def{Key: ReportRepoKey, Label: "Bug report repository", Placeholder: "your-name/cartrak-reports",
			Help: "A PRIVATE GitHub repository that Send a bug report writes to (owner/name). Never your public CarTrak repository: a report holds your screen and your car's readings. Empty: bug reports are off."},
		Def{Key: ReportTokenKey, Label: "Bug report token", Secret: true,
			Help: "A GitHub token limited to that one repository, with Contents and Issues set to Read and write. Stored on the server only; steps are in docs/bug-reports.md."})
	validators[ReportRepoKey] = func(v string) error {
		if v != "" && !reportRepoRE.MatchString(v) {
			return fmt.Errorf("write it as owner/name, such as your-name/cartrak-reports")
		}
		return nil
	}
}

// ReportRepo is the repository bug reports go to ("owner/name"), or "" when they are off.
func (s *Settings) ReportRepo() string { return strings.TrimSpace(s.Value(ReportRepoKey)) }

// ReportToken is the GitHub token for that repository, or "" when none is set.
func (s *Settings) ReportToken() string { return strings.TrimSpace(s.Value(ReportTokenKey)) }
