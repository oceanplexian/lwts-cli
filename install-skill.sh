#!/bin/bash
# Install the clawd skill globally for Claude Code

SKILL_SRC="$(cd "$(dirname "$0")" && pwd)/.claude/skills/clawd"
SKILL_DST="$HOME/.claude/skills/clawd"

if [ ! -f "$SKILL_SRC/SKILL.md" ]; then
  echo "Error: SKILL.md not found at $SKILL_SRC"
  exit 1
fi

mkdir -p "$SKILL_DST"
cp "$SKILL_SRC/SKILL.md" "$SKILL_DST/SKILL.md"
echo "Installed clawd skill to $SKILL_DST"
