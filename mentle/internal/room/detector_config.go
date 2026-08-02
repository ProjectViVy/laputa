package room

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ConfigBasedRoomDetector struct {
	rooms []RoomConfig
}

func NewConfigBasedRoomDetector(projectDir string) (*ConfigBasedRoomDetector, error) {
	rooms := []RoomConfig{
		{Name: "general"},
	}

	path, found := FindConfigFile(projectDir)
	if found {
		loaded, err := LoadRoomsFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load room config: %w", err)
		}
		if len(loaded) > 0 {
			rooms = loaded
		}
	}

	return &ConfigBasedRoomDetector{rooms: rooms}, nil
}

func (d *ConfigBasedRoomDetector) Rooms() []RoomConfig {
	return d.rooms
}

func (d *ConfigBasedRoomDetector) DetectRoom(filePath string, content string, projectPath string) string {
	if len(d.rooms) == 0 {
		return "general"
	}

	relative := strings.TrimPrefix(strings.ToLower(filePath), strings.ToLower(projectPath)+string(os.PathSeparator))
	if relative == "" {
		relative = "."
	}
	filename := strings.ToLower(filepath.Base(filePath))
	contentLen := min(len(content), 2000)
	contentLower := strings.ToLower(content[:contentLen])

	pathParts := strings.Split(strings.ReplaceAll(relative, `\`, `/`), "/")
	if len(pathParts) > 1 {
		for _, part := range pathParts[:len(pathParts)-1] {
			for _, room := range d.rooms {
				if part == strings.ToLower(room.Name) {
					return room.Name
				}
				for _, keyword := range room.Keywords {
					kwLower := strings.ToLower(keyword)
					if strings.Contains(part, kwLower) || strings.Contains(kwLower, part) {
						return room.Name
					}
				}
			}
		}
	}

	for _, room := range d.rooms {
		roomLower := strings.ToLower(room.Name)
		if strings.Contains(filename, roomLower) || strings.Contains(roomLower, filename) {
			return room.Name
		}
	}

	scores := make(map[string]int)
	for _, room := range d.rooms {
		score := 0
		roomLower := strings.ToLower(room.Name)
		score += strings.Count(contentLower, roomLower)
		for _, keyword := range room.Keywords {
			kwLower := strings.ToLower(keyword)
			if kwLower != "" {
				score += strings.Count(contentLower, kwLower)
			}
		}
		if score > 0 {
			scores[room.Name] = score
		}
	}

	if len(scores) > 0 {
		bestRoom := "general"
		bestScore := 0
		for roomName, score := range scores {
			if score > bestScore {
				bestScore = score
				bestRoom = roomName
			}
		}
		if bestScore > 0 {
			return bestRoom
		}
	}

	return "general"
}
