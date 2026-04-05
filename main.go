package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

type Config struct {
	APIURL   string
	APIToken string
}

func loadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	path := filepath.Join(home, ".config", "lwts", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config not found at %s — run: lwts-cli setup", path)
	}

	var cfg Config
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, ":"); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "api_url":
				cfg.APIURL = v
			case "api_token":
				cfg.APIToken = v
			}
		}
	}
	if cfg.APIURL == "" || cfg.APIToken == "" {
		return Config{}, fmt.Errorf("config at %s missing api_url or api_token", path)
	}
	return cfg, nil
}

func (c Config) request(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.APIURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// ── Data types ──────────────────────────────────────────────────────────────

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Initials    string `json:"initials"`
	AvatarColor string `json:"avatar_color"`
}

type Board struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProjectKey string `json:"project_key"`
}

type Card struct {
	ID           string  `json:"id"`
	BoardID      string  `json:"board_id"`
	ColumnID     string  `json:"column_id"`
	Key          string  `json:"key"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Tag          string  `json:"tag"`
	Priority     string  `json:"priority"`
	AssigneeID   *string `json:"assignee_id"`
	AssigneeName string  `json:"assignee_name"`
	ReporterID   *string `json:"reporter_id"`
	Points       *int    `json:"points"`
	DueDate      *string `json:"due_date"`
	Version      int     `json:"version"`
	Position     int     `json:"position"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type Comment struct {
	ID        string `json:"id"`
	CardID    string `json:"card_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// ── Commands ────────────────────────────────────────────────────────────────

func cmdMe(cfg Config) {
	data, err := cfg.request("GET", "/api/auth/me", nil)
	fatal(err)
	var u User
	fatal(json.Unmarshal(data, &u))
	fmt.Printf("%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
}

func cmdUsers(cfg Config) {
	data, err := cfg.request("GET", "/api/v1/users", nil)
	fatal(err)
	var users []User
	fatal(json.Unmarshal(data, &users))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
	}
	w.Flush()
}

func cmdBoards(cfg Config) {
	data, err := cfg.request("GET", "/api/v1/boards", nil)
	fatal(err)
	var boards []Board
	fatal(json.Unmarshal(data, &boards))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKEY")
	for _, b := range boards {
		fmt.Fprintf(w, "%s\t%s\t%s\n", b.ID, b.Name, b.ProjectKey)
	}
	w.Flush()
}

func cmdCards(cfg Config, args []string) {
	boardID := resolveBoardID(cfg, args)
	data, err := cfg.request("GET", "/api/v1/boards/"+boardID+"/cards", nil)
	fatal(err)

	var columns map[string][]Card
	fatal(json.Unmarshal(data, &columns))

	// Build user lookup
	userMap := getUserMap(cfg)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tPRIORITY\tTITLE\tCOLUMN\tASSIGNEE\tPOINTS")

	for _, col := range []string{"backlog", "todo", "in-progress", "done"} {
		cards, ok := columns[col]
		if !ok {
			continue
		}
		for _, c := range cards {
			assignee := resolveAssignee(c, userMap)
			pts := "-"
			if c.Points != nil {
				pts = fmt.Sprintf("%d", *c.Points)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				c.Key, c.Priority, truncate(c.Title, 50), col, assignee, pts)
		}
	}
	w.Flush()
}

func cmdCard(cfg Config, keyOrID string) {
	card := getCard(cfg, keyOrID)
	userMap := getUserMap(cfg)
	assignee := resolveAssignee(card, userMap)
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
	cdata, err := cfg.request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
	if err == nil {
		var comments []Comment
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

func cmdCreate(cfg Config, args []string) {
	if len(args) < 1 {
		fatal(fmt.Errorf("usage: lwts-cli create <title> [--board=ID] [--column=todo] [--tag=blue] [--priority=medium] [--assignee=UUID] [--points=N] [--due=DATE] [--desc=TEXT]"))
	}

	title := args[0]
	flags := parseFlags(args[1:])

	boardID := flags["board"]
	if boardID == "" {
		boardID = resolveBoardID(cfg, nil)
	}

	body := map[string]interface{}{
		"title":     title,
		"column_id": flagOr(flags, "column", "todo"),
		"tag":       mapTag(flagOr(flags, "tag", "blue")),
		"priority":  mapPriority(flagOr(flags, "priority", "medium")),
	}
	if v := flags["assignee"]; v != "" {
		body["assignee_id"] = v
	}
	if v := flags["points"]; v != "" {
		var pts int
		fmt.Sscanf(v, "%d", &pts)
		body["points"] = pts
	}
	if v := flags["due"]; v != "" {
		body["due_date"] = v
	}
	if v := flags["desc"]; v != "" {
		body["description"] = v
	}

	data, err := cfg.request("POST", "/api/v1/boards/"+boardID+"/cards", body)
	fatal(err)
	var card Card
	fatal(json.Unmarshal(data, &card))
	fmt.Printf("created %s: %s\n", card.Key, card.Title)
}

func cmdUpdate(cfg Config, keyOrID string, args []string) {
	card := getCard(cfg, keyOrID)
	flags := parseFlags(args)

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
		body["tag"] = mapTag(v)
	}
	if v := flags["priority"]; v != "" {
		body["priority"] = mapPriority(v)
	}
	if v := flags["assignee"]; v != "" {
		body["assignee_id"] = v
	}
	if v := flags["points"]; v != "" {
		var pts int
		fmt.Sscanf(v, "%d", &pts)
		body["points"] = pts
	}
	if v := flags["due"]; v != "" {
		body["due_date"] = v
	}

	_, err := cfg.request("PUT", "/api/v1/cards/"+card.ID, body)
	fatal(err)
	fmt.Printf("updated %s\n", card.Key)
}

func cmdMove(cfg Config, keyOrID string, column string) {
	card := getCard(cfg, keyOrID)
	body := map[string]interface{}{
		"column_id": column,
		"position":  0,
		"version":   card.Version,
	}
	_, err := cfg.request("POST", "/api/v1/cards/"+card.ID+"/move", body)
	fatal(err)
	fmt.Printf("moved %s → %s\n", card.Key, column)
}

func cmdDelete(cfg Config, keyOrID string) {
	card := getCard(cfg, keyOrID)
	_, err := cfg.request("DELETE", "/api/v1/cards/"+card.ID, nil)
	fatal(err)
	fmt.Printf("deleted %s: %s\n", card.Key, card.Title)
}

func cmdComment(cfg Config, keyOrID string, body string) {
	card := getCard(cfg, keyOrID)
	payload := map[string]string{"body": body}
	_, err := cfg.request("POST", "/api/v1/cards/"+card.ID+"/comments", payload)
	fatal(err)
	fmt.Printf("commented on %s\n", card.Key)
}

func cmdComments(cfg Config, keyOrID string) {
	card := getCard(cfg, keyOrID)
	data, err := cfg.request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
	fatal(err)
	var comments []Comment
	fatal(json.Unmarshal(data, &comments))

	userMap := getUserMap(cfg)
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

func cmdSearch(cfg Config, args []string) {
	flags := parseFlags(args)
	params := url.Values{}
	for _, k := range []string{"q", "assignee", "assignee_id", "column_id", "tag", "priority", "board_id", "limit"} {
		if v := flags[k]; v != "" {
			if k == "tag" {
				v = mapTag(v)
			}
			if k == "priority" {
				v = mapPriority(v)
			}
			params.Set(k, v)
		}
	}

	if len(params) == 0 {
		fatal(fmt.Errorf("search requires at least one filter: --q, --assignee, --column_id, --tag, --priority, --board_id"))
	}

	data, err := cfg.request("GET", "/api/v1/search?"+params.Encode(), nil)
	fatal(err)
	var cards []Card
	fatal(json.Unmarshal(data, &cards))

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
			c.Key, c.Priority, truncate(c.Title, 50), c.ColumnID, assignee, pts)
	}
	w.Flush()

	if len(cards) == 0 {
		fmt.Println("no results")
	}
}

func cmdSetup() {
	home, err := os.UserHomeDir()
	fatal(err)
	dir := filepath.Join(home, ".config", "lwts")
	path := filepath.Join(dir, "config.yaml")

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config already exists at %s\n", path)
		data, _ := os.ReadFile(path)
		fmt.Println(string(data))
		return
	}

	fatal(os.MkdirAll(dir, 0755))
	content := "api_url: https://your-lwts-instance.example.com\napi_token: lwts_sk_your-token-here\n"
	fatal(os.WriteFile(path, []byte(content), 0600))
	fmt.Printf("created %s — edit it with your instance URL and API token\n", path)
	fmt.Println("generate a token with: lwts api-key <your-email>")
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func getCard(cfg Config, keyOrID string) Card {
	// Try by UUID first
	data, err := cfg.request("GET", "/api/v1/cards/"+keyOrID, nil)
	if err == nil {
		var card Card
		if json.Unmarshal(data, &card) == nil && card.ID != "" {
			return card
		}
	}

	// If it looks like a key (e.g. KANB-2), scan all boards
	if strings.Contains(keyOrID, "-") {
		bdata, err := cfg.request("GET", "/api/v1/boards", nil)
		if err == nil {
			var boards []Board
			json.Unmarshal(bdata, &boards)
			for _, b := range boards {
				cdata, err := cfg.request("GET", "/api/v1/boards/"+b.ID+"/cards", nil)
				if err != nil {
					continue
				}
				var columns map[string][]Card
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

	fatal(fmt.Errorf("card not found: %s", keyOrID))
	return Card{} // unreachable
}

func resolveBoardID(cfg Config, args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	data, err := cfg.request("GET", "/api/v1/boards", nil)
	fatal(err)
	var boards []Board
	fatal(json.Unmarshal(data, &boards))

	if len(boards) == 0 {
		fatal(fmt.Errorf("no boards found"))
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

func getUserMap(cfg Config) map[string]string {
	data, err := cfg.request("GET", "/api/v1/users", nil)
	if err != nil {
		return nil
	}
	var users []User
	if json.Unmarshal(data, &users) != nil {
		return nil
	}
	m := make(map[string]string, len(users))
	for _, u := range users {
		m[u.ID] = u.Name
	}
	return m
}

func resolveAssignee(c Card, userMap map[string]string) string {
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

func parseFlags(args []string) map[string]string {
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

func flagOr(flags map[string]string, key, def string) string {
	if v := flags[key]; v != "" {
		return v
	}
	return def
}

func mapTag(s string) string {
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

func mapPriority(s string) string {
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// setup doesn't need config
	if cmd == "setup" {
		cmdSetup()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "me":
		cmdMe(cfg)
	case "users":
		cmdUsers(cfg)
	case "boards":
		cmdBoards(cfg)
	case "cards":
		cmdCards(cfg, os.Args[2:])
	case "card":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: lwts-cli card <key>"))
		}
		cmdCard(cfg, os.Args[2])
	case "create":
		cmdCreate(cfg, os.Args[2:])
	case "update":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: lwts-cli update <key> --field=value ..."))
		}
		cmdUpdate(cfg, os.Args[2], os.Args[3:])
	case "move":
		if len(os.Args) < 4 {
			fatal(fmt.Errorf("usage: lwts-cli move <key> <column>"))
		}
		cmdMove(cfg, os.Args[2], os.Args[3])
	case "delete":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: lwts-cli delete <key>"))
		}
		cmdDelete(cfg, os.Args[2])
	case "comment":
		if len(os.Args) < 4 {
			fatal(fmt.Errorf("usage: lwts-cli comment <key> <body>"))
		}
		cmdComment(cfg, os.Args[2], strings.Join(os.Args[3:], " "))
	case "comments":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: lwts-cli comments <key>"))
		}
		cmdComments(cfg, os.Args[2])
	case "search":
		cmdSearch(cfg, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `lwts-cli — LWTS kanban board CLI

Usage: lwts-cli <command> [args]

Commands:
  setup                          Create config file
  me                             Show current user
  users                          List all users
  boards                         List all boards
  cards [board_id]               List cards (auto-selects if one board)
  card <key>                     Show card detail + comments
  create <title> [flags]         Create a card
  update <key> [flags]           Update a card
  move <key> <column>            Move card to column
  delete <key>                   Delete a card
  comment <key> <body>           Add a comment
  comments <key>                 List comments
  search [flags]                 Search cards

Create/Update flags:
  --board=ID        Board ID
  --column=COL      Column (backlog, todo, in-progress, done)
  --tag=TAG         Tag (blue/feature, green/fix, orange/infra, red/bug)
  --priority=PRI    Priority (highest/critical, high, medium, low, lowest)
  --assignee=UUID   Assignee user ID
  --points=N        Story points
  --due=DATE        Due date (YYYY-MM-DD)
  --desc=TEXT        Description (markdown)
  --title=TEXT      New title (update only)

Search flags:
  --q=TEXT          Search title/description
  --assignee=NAME   Fuzzy match user name
  --assignee_id=ID  Exact user ID
  --column_id=COL   Filter by column
  --tag=TAG         Filter by tag
  --priority=PRI    Filter by priority
  --board_id=ID     Filter by board
  --limit=N         Max results (default 50)

Config: ~/.config/lwts/config.yaml
`)
}
