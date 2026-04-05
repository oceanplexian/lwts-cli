#!/bin/bash
# Install the lwts skill globally for Claude Code

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_SRC="$REPO_DIR/.claude/skills/lwts"
SKILL_DST="$HOME/.claude/skills/lwts"

if [ ! -f "$SKILL_SRC/SKILL.md" ]; then
  echo "Error: SKILL.md not found at $SKILL_SRC"
  exit 1
fi

# Remove old clawd skill if it exists
if [ -d "$HOME/.claude/skills/clawd" ]; then
  rm -rf "$HOME/.claude/skills/clawd"
  echo "Removed old clawd skill"
fi

# Build and install the lwts-cli binary into $GOPATH/bin (or $HOME/go/bin)
echo "Building lwts-cli..."
cd "$REPO_DIR" && go install .
if [ $? -ne 0 ]; then
  echo "Error: failed to build lwts-cli"
  exit 1
fi
echo "Installed lwts-cli to $(go env GOPATH)/bin/lwts-cli"

mkdir -p "$SKILL_DST"
cp "$SKILL_SRC/SKILL.md" "$SKILL_DST/SKILL.md"
echo "Installed lwts skill to $SKILL_DST"
