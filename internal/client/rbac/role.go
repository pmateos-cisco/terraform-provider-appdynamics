package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// Role represents an AppDynamics RBAC role. Permissions is passed through as
// raw JSON since the API echoes each permission back with extra
// server-assigned fields (id, entityId, tagList) that were never in the
// original request (verified live) -- same state-consistency concern as the
// JSON-passthrough attributes in the alertandrespond package. Permissions is
// only ever populated on a GET when include-permissions=true is passed
// (verified live: a plain GET, and the Create response, return only id/name).
type Role struct {
	ID          int64           `json:"id,omitempty"`
	Name        string          `json:"name"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
}

// RoleSummary is the abbreviated representation returned by the roles list
// endpoint (id, name only -- use GetRole for full detail, with permissions).
type RoleSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type roleListEnvelope struct {
	Roles []RoleSummary `json:"roles"`
}

func rolesPath() string {
	return basePath + "/roles"
}

func rolePath(roleID int64) string {
	return fmt.Sprintf("%s/%d", rolesPath(), roleID)
}

func rolePathWithPermissions(roleID int64) string {
	return rolePath(roleID) + "?include-permissions=true"
}

func roleByNamePath(name string) string {
	return fmt.Sprintf("%s/name/%s?include-permissions=true", rolesPath(), url.PathEscape(name))
}

func roleUserPath(roleID, userID int64) string {
	return fmt.Sprintf("%s/users/%d", rolePath(roleID), userID)
}

func roleGroupPath(roleID, groupID int64) string {
	return fmt.Sprintf("%s/groups/%d", rolePath(roleID), groupID)
}

func CreateRole(ctx context.Context, c *client.Client, r *Role) (*Role, error) {
	var out Role
	if err := do(ctx, c, http.MethodPost, rolesPath(), r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRole always requests include-permissions=true so Permissions is populated.
func GetRole(ctx context.Context, c *client.Client, roleID int64) (*Role, error) {
	var out Role
	if err := do(ctx, c, http.MethodGet, rolePathWithPermissions(roleID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func GetRoleByName(ctx context.Context, c *client.Client, name string) (*Role, error) {
	var out Role
	if err := do(ctx, c, http.MethodGet, roleByNamePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func UpdateRole(ctx context.Context, c *client.Client, r *Role) (*Role, error) {
	var out Role
	if err := do(ctx, c, http.MethodPut, rolePath(r.ID), r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func DeleteRole(ctx context.Context, c *client.Client, roleID int64) error {
	return do(ctx, c, http.MethodDelete, rolePath(roleID), nil, nil)
}

func ListRoles(ctx context.Context, c *client.Client) ([]RoleSummary, error) {
	var out roleListEnvelope
	if err := do(ctx, c, http.MethodGet, rolesPath(), nil, &out); err != nil {
		return nil, err
	}
	return out.Roles, nil
}

// AssignRoleToUser/RemoveRoleFromUser and AssignRoleToGroup/RemoveRoleFromGroup
// manage role assignment. None of the four return a body (verified live).
// A role-user assignment is reflected on the user's Roles field; a
// role-group assignment is reflected on the GROUP's Roles field (unlike
// group membership, which is only ever reflected on the user) -- both
// verified live.
func AssignRoleToUser(ctx context.Context, c *client.Client, roleID, userID int64) error {
	return do(ctx, c, http.MethodPut, roleUserPath(roleID, userID), nil, nil)
}

func RemoveRoleFromUser(ctx context.Context, c *client.Client, roleID, userID int64) error {
	return do(ctx, c, http.MethodDelete, roleUserPath(roleID, userID), nil, nil)
}

func AssignRoleToGroup(ctx context.Context, c *client.Client, roleID, groupID int64) error {
	return do(ctx, c, http.MethodPut, roleGroupPath(roleID, groupID), nil, nil)
}

func RemoveRoleFromGroup(ctx context.Context, c *client.Client, roleID, groupID int64) error {
	return do(ctx, c, http.MethodDelete, roleGroupPath(roleID, groupID), nil, nil)
}
