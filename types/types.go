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
}

type Comment struct {
	ID        string `json:"id"`
	CardID    string `json:"card_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}
