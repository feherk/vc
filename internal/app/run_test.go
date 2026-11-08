package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/feherkaroly/vc/internal/model"
)

func TestRunCommandUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix rules")
	}
	cases := []struct {
		name string
		mode os.FileMode
		want string // "" = not runnable
	}{
		{"deploy.sh", 0644, "bash"},
		{"deploy.sh", 0755, "/d/deploy.sh"},
		{"start.command", 0644, "bash"},
		{"x.zsh", 0644, "zsh"},
		{"tool", 0755, "/d/tool"},
		{"tool", 0644, ""},
		{"photo.jpg", 0777, ""}, // exFAT: every file is 0777
		{"doc.pdf", 0755, ""},
		{"script.py", 0755, "/d/script.py"},
		{"script.py", 0644, ""},
	}
	for _, c := range cases {
		e := &model.FileEntry{Name: c.name, Mode: c.mode}
		cmd := runCommand("/d/"+c.name, e)
		got := ""
		if cmd != nil {
			got = cmd.Args[0]
		}
		if got != c.want {
			t.Errorf("%s %o: got %q want %q", c.name, c.mode, got, c.want)
		}
	}
}
