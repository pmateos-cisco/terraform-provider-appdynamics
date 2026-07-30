package rbac

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// Group represents an AppDynamics RBAC group. Unlike User, a group's member
// users are never reflected on the group itself (verified live) -- only on
// each user's own "groups" field -- so Group has no Users field. Roles
// assigned to the group ARE reflected here, though.
type Group struct {
	ID                   int64     `json:"id,omitempty"`
	Name                 string    `json:"name"`
	Description          string    `json:"description,omitempty"`
	SecurityProviderType string    `json:"security_provider_type"`
	Roles                []RoleRef `json:"roles,omitempty"`
}

// GroupSummary is the abbreviated representation returned by the groups list
// endpoint (id, name only -- use GetGroup for full detail).
type GroupSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type groupListEnvelope struct {
	Groups []GroupSummary `json:"groups"`
}

func groupsPath() string {
	return basePath + "/groups"
}

func groupPath(groupID int64) string {
	return fmt.Sprintf("%s/%d", groupsPath(), groupID)
}

func groupByNamePath(name string) string {
	return fmt.Sprintf("%s/name/%s", groupsPath(), url.PathEscape(name))
}

func groupUserPath(groupID, userID int64) string {
	return fmt.Sprintf("%s/users/%d", groupPath(groupID), userID)
}

func CreateGroup(ctx context.Context, c *client.Client, g *Group) (*Group, error) {
	var out Group
	if err := do(ctx, c, http.MethodPost, groupsPath(), g, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func GetGroup(ctx context.Context, c *client.Client, groupID int64) (*Group, error) {
	var out Group
	if err := do(ctx, c, http.MethodGet, groupPath(groupID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func GetGroupByName(ctx context.Context, c *client.Client, name string) (*Group, error) {
	var out Group
	if err := do(ctx, c, http.MethodGet, groupByNamePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func UpdateGroup(ctx context.Context, c *client.Client, g *Group) (*Group, error) {
	var out Group
	if err := do(ctx, c, http.MethodPut, groupPath(g.ID), g, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func DeleteGroup(ctx context.Context, c *client.Client, groupID int64) error {
	return do(ctx, c, http.MethodDelete, groupPath(groupID), nil, nil)
}

func ListGroups(ctx context.Context, c *client.Client) ([]GroupSummary, error) {
	var out groupListEnvelope
	if err := do(ctx, c, http.MethodGet, groupsPath(), nil, &out); err != nil {
		return nil, err
	}
	return out.Groups, nil
}

// AddUserToGroup and RemoveUserFromGroup manage group membership. Neither
// endpoint returns a body (verified live) -- membership must be verified by
// re-reading the user (see GetUser's Groups field), not the group.
func AddUserToGroup(ctx context.Context, c *client.Client, groupID, userID int64) error {
	return do(ctx, c, http.MethodPut, groupUserPath(groupID, userID), nil, nil)
}

func RemoveUserFromGroup(ctx context.Context, c *client.Client, groupID, userID int64) error {
	return do(ctx, c, http.MethodDelete, groupUserPath(groupID, userID), nil, nil)
}
