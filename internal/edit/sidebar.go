package edit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
)

type SideFile struct {
	Path      string
	Name      string
	TitleText string
	DescText  string
}

var cachedProjectRoot string

func (s SideFile) FilterValue() string { return s.Name + " " + s.TitleText + " " + s.DescText }
func (s SideFile) Title() string       { return s.TitleText }
func (s SideFile) Description() string { return s.DescText }
func (s SideFile) String() string      { return fmt.Sprintf("SideFile(%q)", s.Path) }

func FindProjectRoot() string {
	if cachedProjectRoot != "" {
		return cachedProjectRoot
	}
	cwd, err := os.Getwd()
	if err != nil {
		cachedProjectRoot = "."
		return cachedProjectRoot
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		cachedProjectRoot = cwd
		return cachedProjectRoot
	}
	cachedProjectRoot = strings.TrimSpace(string(out))
	return cachedProjectRoot
}

func TrimParents(rel string, maxCells int) string {
	if maxCells < 1 {
		return ""
	}
	rel = strings.TrimPrefix(rel, "./")
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) <= 1 {
		return ""
	}
	dirs := parts[:len(parts)-1]
	n := len(dirs)
	if n == 0 {
		return ""
	}

	sepCost := n - 1
	if 1*n+sepCost > maxCells {
		return trimParentsEllipsis(dirs, maxCells)
	}

	maxPer := max(1, (maxCells-sepCost)/n)
	for {
		cost := sepCost
		for _, d := range dirs {
			cost += min(maxPer, len(d))
		}
		if cost <= maxCells {
			break
		}
		maxPer--
		if maxPer < 1 {
			maxPer = 1
			break
		}
	}

	out := make([]string, n)
	used := sepCost
	for i, d := range dirs {
		take := min(maxPer, len(d))
		out[i] = d[:take]
		used += take
	}

	leftover := maxCells - used
	for i := n - 1; i >= 0 && leftover > 0; i-- {
		if len(out[i]) < len(dirs[i]) {
			out[i] = dirs[i][:len(out[i])+1]
			leftover--
		}
	}

	return strings.Join(out, string(filepath.Separator))
}

func trimParentsEllipsis(dirs []string, maxCells int) string {
	out := make([]string, 0, len(dirs))
	used := 0
	for i := len(dirs) - 1; i >= 0; i-- {
		seg := abbreviateDir(dirs[i])
		cost := len(seg)
		if len(out) > 0 {
			cost++
		}
		if used+cost > maxCells {
			if len(out) > 0 && used+2 <= maxCells {
				out = append([]string{"…"}, out...)
			}
			break
		}
		out = append([]string{seg}, out...)
		used += cost
	}
	return strings.Join(out, string(filepath.Separator))
}

func abbreviateDir(d string) string {
	if d == "" {
		return ""
	}
	runes := []rune(d)
	if strings.HasPrefix(d, ".") {
		if len(runes) == 1 {
			return "."
		}
		return "." + string(runes[1])
	}
	return string(runes[0])
}

func ScanMarkdownFiles(descCells int) ([]list.Item, error) {
	if descCells <= 0 {
		descCells = 22
	}
	root := FindProjectRoot()
	var items []list.Item
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() && info.Name() == "node_modules" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			items = append(items, SideFile{
				Path:      path,
				Name:      rel,
				TitleText: filepath.Base(path),
				DescText:  TrimParents(rel, descCells),
			})
		}
		return nil
	})
	if walkErr != nil {
		return items, walkErr
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].(SideFile).Name < items[j].(SideFile).Name
	})
	return items, nil
}
