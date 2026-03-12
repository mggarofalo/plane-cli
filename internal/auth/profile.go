package auth

import "fmt"

// ProfileManager handles multi-profile operations.
type ProfileManager struct {
	Config *Config
	Store  SecretStore
}

// List returns all profile names.
func (pm *ProfileManager) List() []string {
	names := make([]string, 0, len(pm.Config.Profiles))
	for name := range pm.Config.Profiles {
		names = append(names, name)
	}
	return names
}

// Switch changes the active profile.
func (pm *ProfileManager) Switch(name string) error {
	if _, ok := pm.Config.Profiles[name]; !ok {
		return fmt.Errorf("profile %q does not exist", name)
	}
	pm.Config.ActiveProfile = name
	return pm.Config.Save()
}

// Delete removes a profile and its keyring entries.
func (pm *ProfileManager) Delete(name string) error {
	if name == pm.Config.ActiveProfile {
		return fmt.Errorf("cannot delete the active profile %q; switch to another profile first", name)
	}
	if _, ok := pm.Config.Profiles[name]; !ok {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Remove keyring entries (best-effort)
	_ = pm.Store.Delete(name + "/api-key")
	_ = pm.Store.Delete(name + "/session-token")

	pm.Config.DeleteProfile(name)
	return pm.Config.Save()
}
