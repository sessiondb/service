// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package utils

// Define the available permissions
var (
	PermUsersRead       = "users:read"
	PermUsersWrite      = "users:write"
	PermRolesManage     = "roles:manage"
	PermInstancesRead   = "instances:read"
	PermInstancesManage = "instances:manage"
	PermLogsView        = "logs:view"
	PermApprovalsManage = "approvals:manage"
)

// RolePermissions maps a system Role's Name to an array of RBAC permission strings
var RolePermissions = map[string][]string{
	"Super Admin": {
		PermUsersRead,
		PermUsersWrite,
		PermRolesManage,
		PermInstancesRead,
		PermInstancesManage,
		PermLogsView,
		PermApprovalsManage,
	},
	"Maintainer": {
		PermUsersRead,
		PermInstancesRead,
		PermInstancesManage,
		PermLogsView,
		PermApprovalsManage,
	},
	"Developer": {
		PermUsersRead,
		PermInstancesRead,
	},
	"Analyst": {
		PermInstancesRead,
		PermLogsView,
	},
}

// GetPermissionsForRole returns the explicit list of permission strings for a given role name
func GetPermissionsForRole(roleName string) []string {
	if perms, ok := RolePermissions[roleName]; ok {
		return perms
	}
	// Return empty permissions for unknown roles to fail securely
	return []string{}
}
