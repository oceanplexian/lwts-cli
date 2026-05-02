---
name: lwts
description: Manage LWTS kanban tickets — create, update, move, delete, comment on, list, and search cards. Use when the user asks about tickets, tasks, cards, board status, assignees, or wants to manage their kanban board. Also use when they ask "what are my tickets" or "what is X working on".
argument-hint: [action or topic]
allowed-tools: Bash Read
---

# LWTS Kanban Board Manager

You manage tickets on the LWTS kanban board via the `lwts-cli` tool.

## Setup — Check First

Run `which lwts-cli` to verify installation. If not found:

```bash
go install github.com/oceanplexian/lwts-cli@latest
```

Then check config exists:

```bash
lwts-cli setup
```

If config is missing, the user needs to create `~/.config/lwts/config.yaml` with their `api_url` and `api_token`. Tokens are generated with `lwts api-key <email>` from the LWTS server admin CLI.

## CLI Reference

### Identity & Users

```bash
lwts-cli me                    # Current user (ID, name, email, role)
lwts-cli users                 # All users table
```

### Boards

```bash
lwts-cli boards                # List all boards (ID, name, project key)
```

### Cards

```bash
lwts-cli cards                 # All cards on default board (auto-selects if one board)
lwts-cli cards <board_id>      # Cards on specific board
lwts-cli card <KEY>            # Full detail + comments (e.g. KANB-1, LWTS-14)
```

### Create

```bash
lwts-cli create "Title" [flags]
```

Flags: `--column=todo` `--tag=blue` `--priority=medium` `--assignee=UUID` `--points=N` `--due=YYYY-MM-DD` `--desc="text"` `--board=ID` `--epic=PARENT_UUID`

### Create an Epic

**Always use the dedicated `epic` subcommand — never `create --tag=blue` for an epic.** Epics are a distinct ticket type (tag=`epic`, rendered purple). Filing an epic with the default blue/feature tag has been a recurring mistake.

```bash
lwts-cli epic "Testing & QA — regression suites, integration, chaos"
```

This wrapper:
- Forces `--tag=epic` (overrides any `--tag` you pass).
- Auto-prefixes the title with `EPIC: ` if missing.
- Defaults priority to `high` (override with `--priority=`).
- Accepts the same flags as `create`, including `--epic=PARENT_UUID` for nested epics.

After creating, capture the new card's UUID (use `--json` and parse `.id`) and pass it as `--epic=<UUID>` when creating or updating child tickets:

```bash
EPIC_ID=$(lwts-cli epic "Title" --json | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")
lwts-cli update FNAI-184 --epic=$EPIC_ID
lwts-cli update FNAI-185 --epic=$EPIC_ID
# ... etc
```

The `epic_id` field is the ONLY real link between a child and its epic. Mentioning "Parent epic: FNAI-X" in the description is documentation, not a wired relationship.

### Update

```bash
lwts-cli update <KEY> [flags]
```

Flags: `--title=TEXT` `--desc=TEXT` `--tag=TAG` `--priority=PRI` `--assignee=UUID` `--points=N` `--due=DATE`

The CLI automatically fetches the current version to avoid conflicts.

### Move

```bash
lwts-cli move <KEY> <column>   # Columns: backlog, todo, in-progress, done, cleared
```

### Delete

```bash
lwts-cli delete <KEY>
```

### Comments

```bash
lwts-cli comment <KEY> "comment body"   # Add comment
lwts-cli comments <KEY>                 # List comments
```

### Search

```bash
lwts-cli search [flags]
```

Flags (at least one required):
- `--q=TEXT` — search title/description
- `--assignee=NAME` — fuzzy match user name (e.g., `--assignee="Claude"`)
- `--column_id=COL` — filter by column (backlog, todo, in-progress, done, cleared)
- `--tag=TAG` — filter by tag
- `--priority=PRI` — filter by priority
- `--board_id=ID` — filter by board
- `--limit=N` — max results

## Tag & Priority Aliases

The CLI accepts friendly names and maps them automatically:

**Tags:** feature/feat -> blue, fix/bugfix -> green, infra/ops -> orange, bug/defect -> red, epic/initiative/purple -> epic

**Priority:** critical/urgent/p0 -> highest, high/p1 -> high, medium/p2 -> medium, low/p3 -> low, lowest/p4 -> lowest

## Closing Tickets

When moving a card to `done`, always add a rich summary comment. Gather context from the conversation, git history, and PRs to build the comment. Use markdown formatting — the board renders it.

**Template:**

```
**Resolved** — <one-line summary of what was done>

**Details:**
- <bullet points covering what changed, decisions made, anything notable>

**Links:**
- PR: <GitHub PR URL if one was created, use `gh pr list` or conversation context>
- Commit: <link to relevant commit(s), e.g. https://github.com/org/repo/commit/SHA>
- Related: <any other relevant links — docs, issues, Slack threads, wiki pages>

**How it was tested:**
- <brief description of testing performed>
```

If no PR or commit exists (e.g. a non-code task), skip those links — don't fabricate them. Use `git log --oneline -5` or `gh pr list --state merged --limit 5` to find recent relevant links when the user doesn't provide them explicitly.

## Agent Coordination — CRITICAL

When working on tickets (especially in parallel with other agents), you MUST update ticket status to prevent conflicts:

1. **Starting work** — As soon as you begin actively working on a ticket, immediately move it to `in-progress` (`lwts-cli move <KEY> in-progress`). This signals to other agents that the ticket is claimed. Do this BEFORE writing any code.
2. **Completing work** — As soon as you finish, move the ticket to `done` and add the closing comment. Don't batch these — close each ticket the moment it's done.
3. **Not immediately working** — If you're just planning, discussing, or creating tickets for later, leave them in `todo`. Only move to `in-progress` when you actually start.

This prevents multiple agents from grabbing the same ticket and doing duplicate work.

## Behavior Rules

1. **"my tickets"** — run `lwts-cli me` to get username, then `lwts-cli search --assignee=<username>`
2. **"what's X working on"** — `lwts-cli search --assignee=<name>`
3. **Creating cards** — default to column=todo, tag=blue, priority=medium. Ask for title at minimum. If the card relates to code or a repo, include `**Repo:** <github URL>` as the first line of the description.
4. **Ambiguity** — if unclear which card, ask. If obvious from context, proceed.
5. **Card references** — users may say "KANB-1" or just "1" with context. Use the full key with the CLI.
6. **Closing cards** — always move to done AND add a rich comment (see "Closing Tickets" above). Never close silently.

## Example Interactions

**"what are my tickets?"**
-> `lwts-cli me` -> get username -> `lwts-cli search --assignee=<username>`

**"show me all bugs"**
-> `lwts-cli search --tag=red`

**"what's in progress?"**
-> `lwts-cli search --column_id=in-progress`

**"create a bug for login crash"**
-> `lwts-cli create "Login page crash" --tag=bug --priority=high`

**"move KANB-5 to done"**
-> `lwts-cli move KANB-5 done`

**"add a comment to KANB-3"**
-> `lwts-cli comment KANB-3 "tested and working"`
