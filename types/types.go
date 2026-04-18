package types

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
	EpicID       *string `json:"epic_id"`
	Version      int     `json:"version"`
	Position     int     `json:"position"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`

	// Populated by /api/v1/search only. Zero on other endpoints.
	Score     float64 `json:"score,omitempty"`
	MatchKind string  `json:"match_kind,omitempty"`
	Snippet   string  `json:"snippet,omitempty"`
}

// SearchResult is the JSON shape the CLI emits in --json mode. Bundles the
// aggregate metadata (which the server sends in response headers) alongside
// the results so an agent has a single structured blob to reason about.
type SearchResult struct {
	Results      []Card `json:"results"`
	TotalMatches int    `json:"total_matches"`
	QueryMode    string `json:"query_mode"`
}

type Comment struct {
	ID        string `json:"id"`
	CardID    string `json:"card_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}
