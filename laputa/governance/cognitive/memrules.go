package cognitive

import (
	"bufio"
	"os"
	"strings"
)

type MemRules struct {
	Path    string
	Version string
	Raw     string
	Rules   []Rule
}

type Rule struct {
	ID    string
	Title string
	Text  string
}

func LoadMemRules(path string) (*MemRules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &MemRules{Path: path, Raw: string(data)}
	m.parse()
	return m, nil
}

func (m *MemRules) Reload() error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	m.Raw = string(data)
	m.Rules = nil
	m.parse()
	return nil
}

func (m *MemRules) parse() {
	scanner := bufio.NewScanner(strings.NewReader(m.Raw))
	inFrontMatter := false
	frontMatterDone := false
	var current *Rule

	for scanner.Scan() {
		line := scanner.Text()

		if !frontMatterDone && strings.TrimSpace(line) == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			}
			inFrontMatter = false
			frontMatterDone = true
			continue
		}
		if inFrontMatter {
			if strings.HasPrefix(line, "version:") {
				m.Version = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
			}
			continue
		}

		if strings.HasPrefix(line, "## ") {
			if current != nil {
				m.Rules = append(m.Rules, *current)
			}
			heading := strings.TrimPrefix(line, "## ")
			id, title := parseRuleHeading(heading)
			current = &Rule{ID: id, Title: title}
			continue
		}

		if current != nil && strings.TrimSpace(line) != "" {
			if current.Text != "" {
				current.Text += "\n"
			}
			current.Text += line
		}
	}
	if current != nil {
		m.Rules = append(m.Rules, *current)
	}
}

func parseRuleHeading(heading string) (id, title string) {
	parts := strings.SplitN(heading, " — ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	parts = strings.SplitN(heading, " - ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", strings.TrimSpace(heading)
}
