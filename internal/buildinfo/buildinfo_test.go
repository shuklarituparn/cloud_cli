package buildinfo

import "testing"

func TestCurrent(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	})
	Version, Commit, Date = "v1", "first", "today"
	if got := Current(); got.Version != "v1" || got.Commit != "first" || got.Date != "today" {
		t.Fatalf("Current()= %+v", got)
	}

}
