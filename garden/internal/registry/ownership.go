package registry

// OwnershipLockPath names the sibling lock for an already canonical resource.
func OwnershipLockPath(canonical string) string { return canonical + ".garden-owner.lock" }
