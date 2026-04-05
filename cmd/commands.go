package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/oceanplexian/lwts-cli/client"
	"github.com/oceanplexian/lwts-cli/types"
)

func CmdMe(cfg client.Config) {
	data, err := cfg.Request("GET", "/api/auth/me", nil)
	Fatal(err)
	var u types.User
	Fatal(json.Unmarshal(data, &u))
	fmt.Printf("%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
}

func CmdUsers(cfg client.Config) {
	data, err := cfg.Request("GET", "/api/v1/users", nil)
	Fatal(err)
	var users []types.User
	Fatal(json.Unmarshal(data, &users))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
	}
	w.Flush()
}

func CmdBoards(cfg client.Config) {
	data, err := cfg.Request("GET", "/api/v1/boards", nil)
	Fatal(err)
	var boards []types.Board
	Fatal(json.Unmarshal(data, &boards))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKEY")
	for _, b := range boards {
		fmt.Fprintf(w, "%s\t%s\t%s\n", b.ID, b.Name, b.ProjectKey)
	}
	w.Flush()
}

func CmdCards(cfg client.Config, args []string) {
	boardID := ResolveBoardID(cfg, args)
	data, err := cfg.Request("GET", "/api/v1/boards/"+boardID+"/cards", nil)
	Fatal(err)

	var columns map[string][]types.Card
	Fatal(json.Unmarshal(data, &columns))

	// Build user lookup
	userMap := GetUserMap(cfg)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tPRIORITY\tTITLE\tCOLUMN\tASSIGNEE\tPOINTS")

	for _, col := range []string{"backlog", "todo", "in-progress", "done"} {
		cards, ok := columns[col]
		if !ok {
			continue
		}
		for _, c := range cards {
			assignee := ResolveAssignee(c, userMap)
			pts := "-"
			if c.Points != nil {
				pts = fmt.Sprintf("%d", *c.Points)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				c.Key, c.Priority, Truncate(c.Title, 50), col, assignee, pts)
		}
	}
	w.Flush()
}

func CmdCard(cfg client.Config, keyOrID string) {
	card := GetCard(cfg, keyOrID)
	userMap := GetUserMap(cfg)
	assignee := ResolveAssignee(card, userMap)
	reporter := "-"
	if card.ReporterID != nil {
		if name, ok := userMap[*card.ReporterID]; ok {
			reporter = name
		}
	}
	pts := "-"
	if card.Points != nil {
		pts = fmt.Sprintf("%d", *card.Points)
	}
	due := "-"
	if card.DueDate != nil {
		due = *card.DueDate
	}

	fmt.Printf("Key:       %s\n", card.Key)
	fmt.Printf("Title:     %s\n", card.Title)
	fmt.Printf("Column:    %s\n", card.ColumnID)
	fmt.Printf("Priority:  %s\n", card.Priority)
	fmt.Printf("Tag:       %s\n", card.Tag)
	fmt.Printf("Assignee:  %s\n", assignee)
	fmt.Printf("Reporter:  %s\n", reporter)
	fmt.Printf("Points:    %s\n", pts)
	fmt.Printf("Due:       %s\n", due)
	fmt.Printf("Version:   %d\n", card.Version)
	fmt.Printf("ID:        %s\n", card.ID)

	if card.Description != "" {
		fmt.Printf("\n--- Description ---\n%s\n", card.Description)
	}

	// Fetch comments
	cdata, err := cfg.Request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
	if err == nil {
		var comments []types.Comment
		if json.Unmarshal(cdata, &comments) == nil && len(comments) > 0 {
			fmt.Printf("\n--- Comments (%d) ---\n", len(comments))
			for _, cm := range comments {
				author := cm.AuthorID
				if name, ok := userMap[cm.AuthorID]; ok {
					author = name
				}
				fmt.Printf("[%s] %s: %s\n", cm.CreatedAt, author, cm.Body)
			}
		}
	}
}

func CmdCreate(cfg client.Config, args []string) {
	if len(args) < 1 {
		Fatal(fmt.Errorf("usage: lwts-cli create <title> [--board=ID] [--column=todo] [--tag=blue] [--priority=medium] [--assignee=UUID] [--points=N] [--due=DATE] [--desc=TEXT]"))
	}

	title := args[0]
	flags := ParseFlags(args[1:])

	boardID := flags["board"]
	if boardID == "" {
		boardID = ResolveBoardID(cfg, nil)
	}

	body := map[string]interface{}{
		"title":     title,
		"column_id": FlagOr(flags, "column", "todo"),
		"tag":       MapTag(FlagOr(flags, "tag", "blue")),
		"priority":  MapPriority(FlagOr(flags, "priority", "medium")),
	}
	if v := flags["assignee"]; v != "" {
		body["assignee_id"] = v
	}
	if v := flags["points"]; v != "" {
		var pts int
		if _, err := fmt.Sscanf(v, "%d", &pts); err == nil {
			body["points"] = pts
		}
	}
	if v := flags["due"]; v != "" {
		body["due_date"] = v
	}
	if v := flags["desc"]; v != "" {
		body["description"] = v
	}

	data, err := cfg.Request("POST", "/api/v1/boards/"+boardID+"/cards", body)
	Fatal(err)
	var card types.Card
	Fatal(json.Unmarshal(data, &card))
	fmt.Printf("created %s: %s\n", card.Key, card.Title)
}

func CmdUpdate(cfg client.Config, keyOrID string, args []string) {
	card := GetCard(cfg, keyOrID)
	flags := ParseFlags(args)

	body := map[string]interface{}{
		"version": card.Version,
	}
	if v := flags["title"]; v != "" {
		body["title"] = v
	}
	if v := flags["desc"]; v != "" {
		body["description"] = v
	}
	if v := flags["tag"]; v != "" {
		body["tag"] = MapTag(v)
	}
	if v := flags["priority"]; v != "" {
		body["priority"] = MapPriority(v)
	}
	if v := flags["assignee"]; v != "" {
		body["assignee_id"] = v
	}
	if v := flags["points"]; v != "" {
		var pts int
		if _, err := fmt.Sscanf(v, "%d", &pts); err == nil {
			body["points"] = pts
		}
	}
	if v := flags["due"]; v != "" {
		body["due_date"] = v
	}

	_, err := cfg.Request("PUT", "/api/v1/cards/"+card.ID, body)
	Fatal(err)
	fmt.Printf("updated %s\n", card.Key)
}

func CmdMove(cfg client.Config, keyOrID string, column string) {
	card := GetCard(cfg, keyOrID)
	body := map[string]interface{}{
		"column_id": column,
		"position":  0,
		"version":   card.Version,
	}
	_, err := cfg.Request("POST", "/api/v1/cards/"+card.ID+"/move", body)
	Fatal(err)
	fmt.Printf("moved %s → %s\n", card.Key, column)
}

func CmdDelete(cfg client.Config, keyOrID string) {
	card := GetCard(cfg, keyOrID)
	_, err := cfg.Request("DELETE", "/api/v1/cards/"+card.ID, nil)
	Fatal(err)
	fmt.Printf("deleted %s: %s\n", card.Key, card.Title)
}

func CmdComment(cfg client.Config, keyOrID string, body string) {
	card := GetCard(cfg, keyOrID)
	payload := map[string]string{"body": body}
	_, err := cfg.Request("POST", "/api/v1/cards/"+card.ID+"/comments", payload)
	Fatal(err)
	fmt.Printf("commented on %s\n", card.Key)
}

func CmdComments(cfg client.Config, keyOrID string) {
	card := GetCard(cfg, keyOrID)
	data, err := cfg.Request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
	Fatal(err)
	var comments []types.Comment
	Fatal(json.Unmarshal(data, &comments))

	userMap := GetUserMap(cfg)
	for _, cm := range comments {
		author := cm.AuthorID
		if name, ok := userMap[cm.AuthorID]; ok {
			author = name
		}
		fmt.Printf("[%s] %s: %s\n", cm.CreatedAt, author, cm.Body)
	}
	if len(comments) == 0 {
		fmt.Println("no comments")
	}
}

func CmdSearch(cfg client.Config, args []string) {
	flags := ParseFlags(args)
	params := url.Values{}
	for _, k := range []string{"q", "assignee", "assignee_id", "column_id", "tag", "priority", "board_id", "limit"} {
		if v := flags[k]; v != "" {
			if k == "tag" {
				v = MapTag(v)
			}
			if k == "priority" {
				v = MapPriority(v)
			}
			params.Set(k, v)
		}
	}

	if len(params) == 0 {
		Fatal(fmt.Errorf("search requires at least one filter: --q, --assignee, --column_id, --tag, --priority, --board_id"))
	}

	data, err := cfg.Request("GET", "/api/v1/search?"+params.Encode(), nil)
	Fatal(err)
	var cards []types.Card
	Fatal(json.Unmarshal(data, &cards))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tPRIORITY\tTITLE\tCOLUMN\tASSIGNEE\tPOINTS")
	for _, c := range cards {
		assignee := c.AssigneeName
		if assignee == "" {
			assignee = "-"
		}
		pts := "-"
		if c.Points != nil {
			pts = fmt.Sprintf("%d", *c.Points)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Key, c.Priority, Truncate(c.Title, 50), c.ColumnID, assignee, pts)
	}
	w.Flush()

	if len(cards) == 0 {
		fmt.Println("no results")
	}
}

func CmdSetup() {
	home, err := os.UserHomeDir()
	Fatal(err)
	dir := filepath.Join(home, ".config", "lwts")
	path := filepath.Join(dir, "config.yaml")

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config already exists at %s\n", path)
		data, _ := os.ReadFile(path)
		fmt.Println(string(data))
		return
	}

	Fatal(os.MkdirAll(dir, 0755))
	content := "api_url: https://your-lwts-instance.example.com\napi_token: lwts_sk_your-token-here\n"
	Fatal(os.WriteFile(path, []byte(content), 0600))
	fmt.Printf("created %s — edit it with your instance URL and API token\n", path)
	fmt.Println("generate a token with: lwts api-key <your-email>")
}
