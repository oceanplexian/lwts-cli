package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/oceanplexian/lwts-cli/client"
	"github.com/oceanplexian/lwts-cli/types"
)

func printJSON(v interface{}) {
	out, err := json.MarshalIndent(v, "", "  ")
	Fatal(err)
	fmt.Println(string(out))
}

func CmdMe(cfg client.Config, jsonMode bool) {
	data, err := cfg.Request("GET", "/api/auth/me", nil)
	Fatal(err)
	var u types.User
	Fatal(json.Unmarshal(data, &u))
	if jsonMode {
		printJSON(u)
		return
	}
	fmt.Printf("%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
}

func CmdUsers(cfg client.Config, jsonMode bool) {
	data, err := cfg.Request("GET", "/api/v1/users", nil)
	Fatal(err)
	var users []types.User
	Fatal(json.Unmarshal(data, &users))

	if jsonMode {
		printJSON(users)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Name, u.Email, u.Role)
	}
	w.Flush()
}

func CmdBoards(cfg client.Config, jsonMode bool) {
	data, err := cfg.Request("GET", "/api/v1/boards", nil)
	Fatal(err)
	var boards []types.Board
	Fatal(json.Unmarshal(data, &boards))

	if jsonMode {
		printJSON(boards)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKEY")
	for _, b := range boards {
		fmt.Fprintf(w, "%s\t%s\t%s\n", b.ID, b.Name, b.ProjectKey)
	}
	w.Flush()
}

// ColumnOrder is the canonical display order for board columns.
// "cleared" is the archive state used by the web UI's "Clear done" button —
// cards there are not shown on the kanban board but are still queryable.
var ColumnOrder = []string{"backlog", "todo", "in-progress", "done", "cleared"}

func CmdCards(cfg client.Config, args []string, jsonMode bool) {
	boardID := ResolveBoardID(cfg, args)
	data, err := cfg.Request("GET", "/api/v1/boards/"+boardID+"/cards", nil)
	Fatal(err)

	var columns map[string][]types.Card
	Fatal(json.Unmarshal(data, &columns))

	if jsonMode {
		var all []types.Card
		for _, col := range ColumnOrder {
			all = append(all, columns[col]...)
		}
		printJSON(all)
		return
	}

	userMap := GetUserMap(cfg)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tPRIORITY\tTITLE\tCOLUMN\tASSIGNEE\tPOINTS")

	for _, col := range ColumnOrder {
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

func CmdCard(cfg client.Config, keyOrID string, jsonMode bool) {
	card := GetCard(cfg, keyOrID)

	if jsonMode {
		type cardWithComments struct {
			types.Card
			Comments []types.Comment `json:"comments,omitempty"`
		}
		out := cardWithComments{Card: card}
		cdata, err := cfg.Request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
		if err == nil {
			_ = json.Unmarshal(cdata, &out.Comments)
		}
		printJSON(out)
		return
	}

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
	epic := "-"
	if card.EpicID != nil && *card.EpicID != "" {
		epic = *card.EpicID
	}
	fmt.Printf("Due:       %s\n", due)
	fmt.Printf("Epic:      %s\n", epic)
	fmt.Printf("Version:   %d\n", card.Version)
	fmt.Printf("ID:        %s\n", card.ID)

	if card.Description != "" {
		fmt.Printf("\n--- Description ---\n%s\n", card.Description)
	}

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

func CmdCreate(cfg client.Config, args []string, jsonMode bool) {
	if len(args) < 1 {
		Fatal(fmt.Errorf("usage: lwts-cli create <title> [--board=ID] [--column=todo] [--tag=blue] [--priority=medium] [--assignee=UUID] [--points=N] [--due=DATE] [--desc=TEXT] [--epic=UUID]"))
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
	if v := flags["epic"]; v != "" {
		body["epic_id"] = v
	}

	data, err := cfg.Request("POST", "/api/v1/boards/"+boardID+"/cards", body)
	Fatal(err)
	var card types.Card
	Fatal(json.Unmarshal(data, &card))
	if jsonMode {
		printJSON(card)
		return
	}
	fmt.Printf("created %s: %s\n", card.Key, card.Title)
}

func CmdUpdate(cfg client.Config, keyOrID string, args []string, jsonMode bool) {
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
	if v := flags["epic"]; v != "" {
		body["epic_id"] = v
	}

	_, err := cfg.Request("PUT", "/api/v1/cards/"+card.ID, body)
	Fatal(err)
	if jsonMode {
		printJSON(map[string]string{"status": "updated", "key": card.Key})
		return
	}
	fmt.Printf("updated %s\n", card.Key)
}

func CmdMove(cfg client.Config, keyOrID string, column string, jsonMode bool) {
	card := GetCard(cfg, keyOrID)
	body := map[string]interface{}{
		"column_id": column,
		"position":  0,
		"version":   card.Version,
	}
	_, err := cfg.Request("POST", "/api/v1/cards/"+card.ID+"/move", body)
	Fatal(err)
	if jsonMode {
		printJSON(map[string]string{"status": "moved", "key": card.Key, "column": column})
		return
	}
	fmt.Printf("moved %s → %s\n", card.Key, column)
}

func CmdDelete(cfg client.Config, keyOrID string, jsonMode bool) {
	card := GetCard(cfg, keyOrID)
	_, err := cfg.Request("DELETE", "/api/v1/cards/"+card.ID, nil)
	Fatal(err)
	if jsonMode {
		printJSON(map[string]string{"status": "deleted", "key": card.Key})
		return
	}
	fmt.Printf("deleted %s: %s\n", card.Key, card.Title)
}

func CmdComment(cfg client.Config, keyOrID string, body string, jsonMode bool) {
	card := GetCard(cfg, keyOrID)
	payload := map[string]string{"body": body}
	_, err := cfg.Request("POST", "/api/v1/cards/"+card.ID+"/comments", payload)
	Fatal(err)
	if jsonMode {
		printJSON(map[string]string{"status": "commented", "key": card.Key})
		return
	}
	fmt.Printf("commented on %s\n", card.Key)
}

func CmdComments(cfg client.Config, keyOrID string, jsonMode bool) {
	card := GetCard(cfg, keyOrID)
	data, err := cfg.Request("GET", "/api/v1/cards/"+card.ID+"/comments", nil)
	Fatal(err)
	var comments []types.Comment
	Fatal(json.Unmarshal(data, &comments))

	if jsonMode {
		printJSON(comments)
		return
	}

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

// CmdSearch queries /api/v1/search with agent-friendly defaults:
//   - limit 5 (not 50): agents rarely need more, reduces token burn
//   - min_score 0.5: drop weak semantic neighbors unless caller asks for them
//   - include_done false: only active work by default
//
// All three can be overridden via flags. Text output is one line per result
// plus an indented snippet; --json emits a stable {results, total_matches,
// query_mode} envelope for programmatic callers.
func CmdSearch(cfg client.Config, args []string, jsonMode bool) {
	flags := ParseFlags(args)
	params := url.Values{}
	for _, k := range []string{"q", "assignee", "assignee_id", "column_id", "tag", "priority", "board_id"} {
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

	// Agent-tuned defaults. The web UI uses the API's own defaults (limit 50,
	// include_done true, no min_score) and isn't affected.
	limit := "5"
	if v := flags["limit"]; v != "" {
		limit = v
	}
	params.Set("limit", limit)

	if v := flags["min_score"]; v != "" {
		params.Set("min_score", v)
	} else if flags["min-score"] != "" {
		params.Set("min_score", flags["min-score"])
	} else {
		params.Set("min_score", "0.5")
	}

	// include_done false unless --include-done or --include_done flag is set.
	includeDone := flags["include_done"] == "true" || flags["include-done"] == "true"
	if !includeDone {
		params.Set("include_done", "false")
	}

	if params.Get("q") == "" && params.Get("assignee") == "" && params.Get("assignee_id") == "" &&
		params.Get("column_id") == "" && params.Get("tag") == "" && params.Get("priority") == "" &&
		params.Get("board_id") == "" {
		Fatal(fmt.Errorf("search requires at least one filter: --q, --assignee, --column_id, --tag, --priority, --board_id"))
	}

	data, hdrs, err := cfg.RequestWithHeaders("GET", "/api/v1/search?"+params.Encode(), nil)
	Fatal(err)
	var cards []types.Card
	Fatal(json.Unmarshal(data, &cards))

	totalMatches := len(cards)
	if hv := hdrs.Get("X-Total-Matches"); hv != "" {
		if n, err := strconv.Atoi(hv); err == nil {
			totalMatches = n
		}
	}
	queryMode := hdrs.Get("X-Search-Mode")
	if queryMode == "" {
		queryMode = "lexical"
	}

	if jsonMode {
		printJSON(types.SearchResult{
			Results:      cards,
			TotalMatches: totalMatches,
			QueryMode:    queryMode,
		})
		return
	}

	if len(cards) == 0 {
		fmt.Println("no results")
		return
	}

	// Human-readable text mode: one line per result plus an indented snippet.
	// Keeps a row scannable while surfacing the "why matched" context an
	// agent (or human skimming) uses to decide whether to drill in.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tCOLUMN\tPRI\tTIER\tKIND\tTITLE")
	for _, c := range cards {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Key, c.ColumnID, shortPriority(c.Priority), scoreTier(c.Score),
			shortKind(c.MatchKind), Truncate(c.Title, 60))
	}
	w.Flush()
	for _, c := range cards {
		if c.Snippet == "" {
			continue
		}
		fmt.Printf("  %s ↳ %s\n", c.Key, c.Snippet)
	}

	if totalMatches > len(cards) {
		fmt.Printf("\n(%d total — %d shown; refine with --priority, --column_id, or --min_score)\n",
			totalMatches, len(cards))
	}
	if queryMode != "" && queryMode != "lexical" {
		fmt.Printf("search mode: %s\n", queryMode)
	}
}

// scoreTier bundles the raw score into a short label an agent can reason
// about at a glance: HIGH is confident, MED is worth reading the snippet, LOW
// means the query was probably too vague.
func scoreTier(score float64) string {
	switch {
	case score >= 0.7:
		return "HIGH"
	case score >= 0.55:
		return "MED"
	default:
		return "LOW"
	}
}

// shortKind abbreviates match_kind for table output.
func shortKind(k string) string {
	switch k {
	case "title_boundary":
		return "title"
	case "semantic":
		return "sem"
	case "lexical":
		return "lex"
	}
	return "-"
}

// shortPriority shortens common priority values so the table stays narrow.
func shortPriority(p string) string {
	switch p {
	case "highest":
		return "P0"
	case "high":
		return "P1"
	case "medium":
		return "P2"
	case "low":
		return "P3"
	case "lowest":
		return "P4"
	}
	return p
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
