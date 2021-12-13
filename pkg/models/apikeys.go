package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type APIKey struct {
	ID              uuid.UUID       `json:"id"`
	Created         time.Time       `json:"created"`
	LastUsed        pq.NullTime     `json:"last_used"`
	PermissionLevel PermissionLevel `json:"permission_level"`
	Description     string          `json:"description"`
	GitHubUser      string          `json:"github_user"`
	Token			uuid.UUID		`json:"token"`
}

func (apik APIKey) Columns() string {
	return strings.Join([]string{"id", "created", "last_used", "permission_level", "description", "github_user", "token"}, ",")
}

func (apik APIKey) InsertColumns() string {
	return strings.Join([]string{"id", "created", "last_used", "permission_level", "description", "github_user", "token"}, ",")
}

func (apik *APIKey) ScanValues() []interface{} {
	return []interface{}{&apik.ID, &apik.Created, &apik.LastUsed, &apik.PermissionLevel, &apik.Description, &apik.GitHubUser, &apik.Token}
}

func (apik *APIKey) InsertValues() []interface{} {
	return []interface{}{&apik.ID, &apik.Created, &apik.LastUsed, &apik.PermissionLevel, &apik.Description, &apik.GitHubUser, &apik.Token}
}

func (apik APIKey) InsertParams() string {
	params := []string{}
	for i := range strings.Split(apik.InsertColumns(), ",") {
		params = append(params, fmt.Sprintf("$%v", i+1))
	}
	return strings.Join(params, ", ")
}

//go:generate stringer -type=PermissionLevel

type PermissionLevel int

const (
	UnknownPermission PermissionLevel = iota
	ReadOnlyPermission
	WritePermission
	AdminPermission
)
