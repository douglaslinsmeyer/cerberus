package auth

import "fmt"

// Role level constants
const (
	RoleViewer      = 1
	RoleContributor = 2
	RoleAdmin       = 3
)

// Role name constants
const (
	RoleNameViewer      = "viewer"
	RoleNameContributor = "contributor"
	RoleNameAdmin       = "admin"
)

// RoleFromString converts a role string to its numeric level
func RoleFromString(role string) (int, error) {
	switch role {
	case RoleNameViewer:
		return RoleViewer, nil
	case RoleNameContributor:
		return RoleContributor, nil
	case RoleNameAdmin:
		return RoleAdmin, nil
	default:
		return 0, fmt.Errorf("invalid role: %s", role)
	}
}

// RoleToString converts a numeric role level to its string name
func RoleToString(level int) (string, error) {
	switch level {
	case RoleViewer:
		return RoleNameViewer, nil
	case RoleContributor:
		return RoleNameContributor, nil
	case RoleAdmin:
		return RoleNameAdmin, nil
	default:
		return "", fmt.Errorf("invalid role level: %d", level)
	}
}

// RoleIsAuthorized checks if userRole meets the minimum required role
func RoleIsAuthorized(userRole string, requiredRole int) (bool, error) {
	userLevel, err := RoleFromString(userRole)
	if err != nil {
		return false, err
	}

	return userLevel >= requiredRole, nil
}

// ValidateRole checks if a role string is valid
func ValidateRole(role string) error {
	_, err := RoleFromString(role)
	return err
}

// Organization role level constants
const (
	OrgRoleMember = 1
	OrgRoleAdmin  = 2
	OrgRoleOwner  = 3
)

// Organization role name constants
const (
	OrgRoleNameMember = "member"
	OrgRoleNameAdmin  = "admin"
	OrgRoleNameOwner  = "owner"
)

// OrgRoleFromString converts an org role string to its numeric level
func OrgRoleFromString(role string) (int, error) {
	switch role {
	case OrgRoleNameMember:
		return OrgRoleMember, nil
	case OrgRoleNameAdmin:
		return OrgRoleAdmin, nil
	case OrgRoleNameOwner:
		return OrgRoleOwner, nil
	default:
		return 0, fmt.Errorf("invalid org role: %s", role)
	}
}

// OrgRoleToString converts a numeric org role level to its string name
func OrgRoleToString(level int) (string, error) {
	switch level {
	case OrgRoleMember:
		return OrgRoleNameMember, nil
	case OrgRoleAdmin:
		return OrgRoleNameAdmin, nil
	case OrgRoleOwner:
		return OrgRoleNameOwner, nil
	default:
		return "", fmt.Errorf("invalid org role level: %d", level)
	}
}

// ValidateOrgRole checks if an org role string is valid
func ValidateOrgRole(role string) error {
	_, err := OrgRoleFromString(role)
	return err
}
