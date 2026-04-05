#!/bin/bash
# Install the lwts skill globally for Claude Code

SKILL_SRC="$(cd "$(dirname "$0")" && pwd)/.claude/skills/lwts"
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

mkdir -p "$SKILL_DST"
cp "$SKILL_SRC/SKILL.md" "$SKILL_DST/SKILL.md"
echo "Installed lwts skill to $SKILL_DST"
