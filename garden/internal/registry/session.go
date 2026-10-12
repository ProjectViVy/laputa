package registry

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

func SessionKey(profileID, bindingID, sessionID string) (string, error) {
	for _, part := range []string{profileID, bindingID, sessionID} {
		if strings.TrimSpace(part) == "" {
			return "", errors.New("session identity requires profile, binding and session")
		}
	}
	tuple := fmt.Sprintf("%d:%s%d:%s%d:%s", len(profileID), profileID, len(bindingID), bindingID, len(sessionID), sessionID)
	return "garden-session/v1:" + base64.RawURLEncoding.EncodeToString([]byte(tuple)), nil
}
