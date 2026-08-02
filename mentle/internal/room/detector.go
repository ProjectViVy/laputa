// Package room provides room detection from project folder structures.
// It maps directory names to semantic room categories.
package room

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var folderRoomMap = map[string]string{
	"frontend":       "frontend",
	"front-end":      "frontend",
	"front_end":      "frontend",
	"client":         "frontend",
	"ui":             "frontend",
	"views":          "frontend",
	"components":     "frontend",
	"pages":          "frontend",
	"backend":        "backend",
	"back-end":       "backend",
	"back_end":       "backend",
	"server":         "backend",
	"api":            "backend",
	"routes":         "backend",
	"services":       "backend",
	"controllers":    "backend",
	"models":         "backend",
	"database":       "backend",
	"db":             "backend",
	"docs":           "documentation",
	"doc":            "documentation",
	"documentation":  "documentation",
	"wiki":           "documentation",
	"readme":         "documentation",
	"notes":          "documentation",
	"design":         "design",
	"designs":        "design",
	"mockups":        "design",
	"wireframes":     "design",
	"assets":         "design",
	"storyboard":     "design",
	"costs":          "costs",
	"cost":           "costs",
	"budget":         "costs",
	"finance":        "costs",
	"financial":      "costs",
	"pricing":        "costs",
	"invoices":       "costs",
	"accounting":     "costs",
	"meetings":       "meetings",
	"meeting":        "meetings",
	"calls":          "meetings",
	"meeting_notes":  "meetings",
	"standup":        "meetings",
	"minutes":        "meetings",
	"team":           "team",
	"staff":          "team",
	"hr":             "team",
	"hiring":         "team",
	"employees":      "team",
	"people":         "team",
	"research":       "research",
	"references":     "research",
	"reading":        "research",
	"papers":         "research",
	"planning":       "planning",
	"roadmap":        "planning",
	"strategy":       "planning",
	"specs":          "planning",
	"requirements":   "planning",
	"tests":          "testing",
	"test":           "testing",
	"testing":        "testing",
	"qa":             "testing",
	"scripts":        "scripts",
	"tools":          "scripts",
	"utils":          "scripts",
	"config":         "configuration",
	"configs":        "configuration",
	"settings":       "configuration",
	"infrastructure": "configuration",
	"infra":          "configuration",
	"deploy":         "configuration",
}

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, "env": true,
	"dist": true, "build": true, ".next": true, "coverage": true,
}

type Room struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

func (d *Detector) DetectRoomsFromFolders(projectDir string) []Room {
	projectPath, err := filepath.Abs(filepath.Clean(projectDir))
	if err != nil {
		return []Room{{Name: "general", Description: "All project files", Keywords: []string{}}}
	}

	foundRooms := make(map[string]string)

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return []Room{{Name: "general", Description: "All project files", Keywords: []string{}}}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if skipDirs[name] {
			continue
		}

		nameLower := strings.ToLower(name)
		nameClean := strings.ReplaceAll(strings.ReplaceAll(nameLower, "-", "_"), " ", "_")

		if roomName, ok := folderRoomMap[nameClean]; ok {
			if _, exists := foundRooms[roomName]; !exists {
				foundRooms[roomName] = name
			}
		} else if len(name) > 2 && unicode.IsLetter(rune(name[0])) {
			clean := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), "-", "_"), " ", "_")
			if _, exists := foundRooms[clean]; !exists {
				foundRooms[clean] = name
			}
		}
	}

	var rooms []Room
	for roomName, original := range foundRooms {
		rooms = append(rooms, Room{
			Name:        roomName,
			Description: "Files from " + original + "/",
			Keywords:    []string{roomName, strings.ToLower(original)},
		})
	}

	if len(rooms) == 0 {
		rooms = append(rooms, Room{
			Name:        "general",
			Description: "Files that don't fit other rooms",
			Keywords:    []string{},
		})
	}

	return rooms
}

func (d *Detector) CountFiles(projectDir string) int {
	count := 0
	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && !skipDirs[filepath.Base(path)] {
			count++
		}
		return nil
	})
	return count
}
