package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var sshUserPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,31}$`)

func LoadUsers(path string) (map[string]BoxRoute, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse users file: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("users file contains no users")
	}
	users := make(map[string]BoxRoute, len(raw))
	for user, value := range raw {
		if !sshUserPattern.MatchString(user) {
			return nil, fmt.Errorf("invalid SSH username %q", user)
		}
		var box string
		if err := json.Unmarshal(value, &box); err == nil {
			if strings.TrimSpace(box) == "" {
				return nil, fmt.Errorf("empty Box for SSH user %q", user)
			}
			users[user] = BoxRoute{Box: strings.TrimSpace(box)}
			continue
		}
		var route BoxRoute
		if err := json.Unmarshal(value, &route); err != nil {
			return nil, fmt.Errorf("invalid route for SSH user %q: %w", user, err)
		}
		if strings.TrimSpace(route.Box) == "" {
			return nil, fmt.Errorf("empty Box for SSH user %q", user)
		}
		users[user] = route
	}
	return users, nil
}

func configHasFlag(args []string, name string) bool {
	prefix := "--" + name
	for _, arg := range args {
		if arg == prefix || strings.HasPrefix(arg, prefix+"=") {
			return true
		}
	}
	return false
}
