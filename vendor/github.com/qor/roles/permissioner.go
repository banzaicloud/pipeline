package roles

// Permissioner permissioner interface
type Permissioner interface {
	HasPermission(mode PermissionMode, roles ...interface{}) bool
}

// ConcatPermissioner concat permissioner
func ConcatPermissioner(ps ...Permissioner) Permissioner {
	var newPS []Permissioner
	for _, p := range ps {
		if p != nil {
			newPS = append(newPS, p)
		}
	}
	return permissioners(newPS)
}

type permissioners []Permissioner

// HasPermission check has permission for permissioners or not
func (ps permissioners) HasPermission(mode PermissionMode, roles ...interface{}) bool {
	for _, p := range ps {
		if p != nil && !p.HasPermission(mode, roles) {
			return false
		}
	}

	return true
}
