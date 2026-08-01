package auth

import "egger/api/internal/store"

// Permission enumera as ações controláveis por RBAC (Seção 18): "Separar
// permissões: Visualizar; Executar testes; Administrar integrações; Administrar
// usuários; Alterar configurações; Executar automações; Exportar dados; Ver
// auditoria."
type Permission string

const (
	PermView               Permission = "view"
	PermRunTests           Permission = "run_tests"
	PermManageIntegrations Permission = "manage_integrations"
	PermManageUsers        Permission = "manage_users"
	PermChangeSettings     Permission = "change_settings"
	PermRunAutomations     Permission = "run_automations"
	PermExportData         Permission = "export_data"
	PermViewAudit          Permission = "view_audit"
)

// rolePermissions define, de forma explícita e testável, o que cada papel pode
// fazer. Owner/Administrator têm superset; Viewer só visualiza; Auditor só
// visualiza + auditoria; Operator executa operação do dia a dia sem administrar
// usuários/integrações.
var rolePermissions = map[store.Role]map[Permission]bool{
	store.RoleOwner: {
		PermView: true, PermRunTests: true, PermManageIntegrations: true,
		PermManageUsers: true, PermChangeSettings: true, PermRunAutomations: true,
		PermExportData: true, PermViewAudit: true,
	},
	store.RoleAdministrator: {
		PermView: true, PermRunTests: true, PermManageIntegrations: true,
		PermManageUsers: true, PermChangeSettings: true, PermRunAutomations: true,
		PermExportData: true, PermViewAudit: true,
	},
	store.RoleOperator: {
		PermView: true, PermRunTests: true, PermRunAutomations: true, PermExportData: true,
	},
	store.RoleViewer: {
		PermView: true,
	},
	store.RoleAuditor: {
		PermView: true, PermViewAudit: true,
	},
}

// HasPermission responde se um papel tem uma permissão. Papel desconhecido nunca
// tem permissão (fail closed).
func HasPermission(role store.Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}
