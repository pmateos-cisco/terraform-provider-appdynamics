package rbac

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// GroupRef and RoleRef identify a group/role a user belongs to, as reflected
// back on the user's own detail response (verified live: membership is
// visible on the user, not on the group; role assignment is visible on
// both the user and the group).
type GroupRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type RoleRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// User represents an AppDynamics RBAC user. Password is write-only: the API
// never returns it in any response (verified live), so it can only be set on
// create/update, never read back.
type User struct {
	ID                   int64      `json:"id,omitempty"`
	Name                 string     `json:"name"`
	DisplayName          string     `json:"displayName,omitempty"`
	SecurityProviderType string     `json:"security_provider_type"`
	Password             string     `json:"password,omitempty"`
	Groups               []GroupRef `json:"groups,omitempty"`
	Roles                []RoleRef  `json:"roles,omitempty"`
}

// UserSummary is the abbreviated representation returned by the users list
// endpoint (id, name only -- use GetUser for full detail).
type UserSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type userListEnvelope struct {
	Users []UserSummary `json:"users"`
}

func usersPath() string {
	return basePath + "/users"
}

func userPath(userID int64) string {
	return fmt.Sprintf("%s/%d", usersPath(), userID)
}

func userByNamePath(name string) string {
	return fmt.Sprintf("%s/name/%s", usersPath(), url.PathEscape(name))
}

func CreateUser(ctx context.Context, c *client.Client, u *User) (*User, error) {
	var out User
	if err := do(ctx, c, http.MethodPost, usersPath(), u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func GetUser(ctx context.Context, c *client.Client, userID int64) (*User, error) {
	var out User
	if err := do(ctx, c, http.MethodGet, userPath(userID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func GetUserByName(ctx context.Context, c *client.Client, name string) (*User, error) {
	var out User
	if err := do(ctx, c, http.MethodGet, userByNamePath(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUser requires ID, Name, and SecurityProviderType to all be set on u
// even though only Name/DisplayName are documented as updatable (verified
// live: the API rejects the request with "id is not match" or
// "'security_provider_type' need to be internal" if either is omitted).
// Name itself is mutable (verified live), so it is not RequiresReplace in
// the resource schema.
func UpdateUser(ctx context.Context, c *client.Client, u *User) (*User, error) {
	var out User
	if err := do(ctx, c, http.MethodPut, userPath(u.ID), u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func DeleteUser(ctx context.Context, c *client.Client, userID int64) error {
	return do(ctx, c, http.MethodDelete, userPath(userID), nil, nil)
}

func ListUsers(ctx context.Context, c *client.Client) ([]UserSummary, error) {
	var out userListEnvelope
	if err := do(ctx, c, http.MethodGet, usersPath(), nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}
