package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/oceanplexian/lwts-cli/client"
	"github.com/oceanplexian/lwts-cli/types"
)

func Fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func GetCard(cfg client.Config, keyOrID string) types.Card {
	// Try by UUID first
	data, err := cfg.Request("GET", "/api/v1/cards/"+keyOrID, nil)
	if err == nil {
		var card types.Card
		if json.Unmarshal(data, &card) == nil && card.ID != "" {
			return card
		}
	}

	// If it looks like a key (e.g. KANB-2), scan all boards
	if strings.Contains(keyOrID, "-") {
		bdata, err := cfg.Request("GET", "/api/v1/boards", nil)
		if err == nil {
			var boards []types.Board
			json.Unmarshal(bdata, &boards)
			for _, b := range boards {
				cdata, err := cfg.Request("GET", "/api/v1/boards/"+b.ID+"/cards", nil)
				if err != nil {
					continue
				}
				var columns map[string][]types.Card
				json.Unmarshal(cdata, &columns)
				for _, cards := range columns {
					for _, c := range cards {
						if strings.EqualFold(c.Key, keyOrID) {
							return c
						}
					}
				}
			}
		}
	}

	Fatal(fmt.Errorf("card not found: %s", keyOrID))
	return types.Card{} // unreachable
}

func ResolveBoardID(cfg client.Config, args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	data, err := cfg.Request("GET", "/api/v1/boards", nil)
	Fatal(err)
	var boards []types.Board
	Fatal(json.Unmarshal(data, &boards))

	if len(boards) == 0 {
		Fatal(fmt.Errorf("no boards found"))
	}
	if len(boards) == 1 {
		return boards[0].ID
	}

	// Multiple boards — print them and error
	fmt.Fprintln(os.Stderr, "multiple boards found, specify --board=ID:")
	for _, b := range boards {
		fmt.Fprintf(os.Stderr, "  %s  %s  %s\n", b.ID, b.Name, b.ProjectKey)
	}
	os.Exit(1)
	return ""
}

func GetUserMap(cfg client.Config) map[string]string {
	data, err := cfg.Request("GET", "/api/v1/users", nil)
	if err != nil {
		return nil
	}
	var users []types.User
	if json.Unmarshal(data, &users) != nil {
		return nil
	}
	m := make(map[string]string, len(users))
	for _, u := range users {
		m[u.ID] = u.Name
	}
	return m
}

func ResolveAssignee(c types.Card, userMap map[string]string) string {
	if c.AssigneeName != "" {
		return c.AssigneeName
	}
	if c.AssigneeID != nil {
		if name, ok := userMap[*c.AssigneeID]; ok {
			return name
		}
	}
	return "-"
}

func ParseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			if k, v, ok := strings.Cut(arg[2:], "="); ok {
				flags[k] = v
			}
		}
	}
	return flags
}

func FlagOr(flags map[string]string, key, def string) string {
	if v := flags[key]; v != "" {
		return v
	}
	return def
}

func MapTag(s string) string {
	switch strings.ToLower(s) {
	case "feature", "feat":
		return "blue"
	case "fix", "bugfix":
		return "green"
	case "infra", "infrastructure", "ops":
		return "orange"
	case "bug", "defect":
		return "red"
	default:
		return s
	}
}

func MapPriority(s string) string {
	switch strings.ToLower(s) {
	case "critical", "urgent", "p0":
		return "highest"
	case "high", "important", "p1":
		return "high"
	case "medium", "normal", "p2":
		return "medium"
	case "low", "minor", "p3":
		return "low"
	case "lowest", "trivial", "p4":
		return "lowest"
	default:
		return s
	}
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
