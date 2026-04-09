package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/oceanplexian/lwts-cli/client"
	"github.com/oceanplexian/lwts-cli/cmd"
)

var version = "dev"

func main() {
	// Detect and strip --json flag from args
	var jsonMode bool
	var filtered []string
	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			jsonMode = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	os.Args = append([]string{os.Args[0]}, filtered...)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// commands that don't need config
	switch command {
	case "setup":
		cmd.CmdSetup()
		return
	case "version":
		fmt.Println(version)
		return
	}

	cfg, err := client.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "me":
		cmd.CmdMe(cfg, jsonMode)
	case "users":
		cmd.CmdUsers(cfg, jsonMode)
	case "boards":
		cmd.CmdBoards(cfg, jsonMode)
	case "cards":
		cmd.CmdCards(cfg, os.Args[2:], jsonMode)
	case "card":
		if len(os.Args) < 3 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli card <key>"))
		}
		cmd.CmdCard(cfg, os.Args[2], jsonMode)
	case "create":
		cmd.CmdCreate(cfg, os.Args[2:], jsonMode)
	case "update":
		if len(os.Args) < 3 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli update <key> --field=value ..."))
		}
		cmd.CmdUpdate(cfg, os.Args[2], os.Args[3:], jsonMode)
	case "move":
		if len(os.Args) < 4 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli move <key> <column>"))
		}
		cmd.CmdMove(cfg, os.Args[2], os.Args[3], jsonMode)
	case "delete":
		if len(os.Args) < 3 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli delete <key>"))
		}
		cmd.CmdDelete(cfg, os.Args[2], jsonMode)
	case "comment":
		if len(os.Args) < 4 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli comment <key> <body>"))
		}
		cmd.CmdComment(cfg, os.Args[2], strings.Join(os.Args[3:], " "), jsonMode)
	case "comments":
		if len(os.Args) < 3 {
			cmd.Fatal(fmt.Errorf("usage: lwts-cli comments <key>"))
		}
		cmd.CmdComments(cfg, os.Args[2], jsonMode)
	case "search":
		cmd.CmdSearch(cfg, os.Args[2:], jsonMode)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `lwts-cli — LWTS kanban board CLI

Usage: lwts-cli <command> [args]

Commands:
  version                        Print version
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

Global flags:
  --json                         Output in JSON format

Create/Update flags:
  --board=ID        Board ID
  --column=COL      Column (backlog, todo, in-progress, done, cleared)
  --tag=TAG         Tag (blue/feature, green/fix, orange/infra, red/bug)
  --priority=PRI    Priority (highest/critical, high, medium, low, lowest)
  --assignee=UUID   Assignee user ID
  --points=N        Story points
  --due=DATE        Due date (YYYY-MM-DD)
  --desc=TEXT        Description (markdown)
  --epic=UUID       Epic card ID
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
