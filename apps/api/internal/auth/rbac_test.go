package auth

import (
	"testing"

	"egger/api/internal/store"
)

func TestHasPermission_Matrix(t *testing.T) {
	cases := []struct {
		role  store.Role
		perm  Permission
		allow bool
	}{
		{store.RoleOwner, PermManageUsers, true},
		{store.RoleAdministrator, PermManageUsers, true},
		{store.RoleOperator, PermManageUsers, false},
		{store.RoleViewer, PermManageUsers, false},
		{store.RoleAuditor, PermManageUsers, false},

		{store.RoleViewer, PermView, true},
		{store.RoleAuditor, PermView, true},
		{store.RoleAuditor, PermViewAudit, true},
		{store.RoleViewer, PermViewAudit, false},

		{store.RoleOperator, PermRunTests, true},
		{store.RoleViewer, PermRunTests, false},

		{store.RoleOperator, PermManageIntegrations, false},
		{store.RoleAdministrator, PermManageIntegrations, true},
	}

	for _, c := range cases {
		got := HasPermission(c.role, c.perm)
		if got != c.allow {
			t.Errorf("HasPermission(%s, %s) = %v, esperado %v", c.role, c.perm, got, c.allow)
		}
	}
}

func TestHasPermission_UnknownRoleFailsClosed(t *testing.T) {
	if HasPermission(store.Role("papel-que-nao-existe"), PermView) {
		t.Fatalf("papel desconhecido nunca deve ter permissão (fail closed)")
	}
}
