package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v2"
)

const global = "global"

// CheckOptions is a map of options for a single check
type CheckOptions map[string]any

// CloudOptions is a map of check names to CheckOptions
type CloudOptions map[string]CheckOptions

// Settings is the top-level configuration structure used for settings.yaml
type Settings struct {
	Default CloudOptions            `yaml:"default"`
	Clouds  map[string]CloudOptions `yaml:"clouds"`
}

// LoadSettingsFromFile loads a settings.yaml file from the given path and returns a Settings struct
func LoadSettingsFromFile(path string) (*Settings, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var settings Settings
	err = yaml.Unmarshal(b, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// GetCloudOptions returns a CloudOptions struct for the given cloud name
func (s *Settings) GetCloudOptions(cloud string) CloudOptions {
	// first set hard-coded default
	defaultGlobalOpts := CheckOptions{
		"interval": 60,
		"timeout":  10,
	}

	// then overlay global defaults from the settings file
	for opt := range s.Default[global] {
		defaultGlobalOpts[opt] = s.Default[global][opt]
	}

	cloudOpts := make(CloudOptions)
	for check, opts := range s.Default {
		if check == global {
			continue
		}

		// make a copy of the global defaults
		cloudOpts[check] = make(CheckOptions)
		for key, value := range defaultGlobalOpts {
			cloudOpts[check][key] = value
		}

		// then overlay the per-check defaults
		for key, value := range opts {
			cloudOpts[check][key] = value
		}
	}

	// then overlay the per-cloud settings
	for check, opts := range s.Clouds[cloud] {
		for key, value := range opts {
			cloudOpts[check][key] = value
		}
	}

	return cloudOpts
}

// Dump prints the settings to stdout
func (opts CloudOptions) Dump() {
	for check, checkopts := range opts {
		fmt.Printf("%s:\n", check)

		var keys []string
		for key := range checkopts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s: %v\n", k, checkopts[k])
		}
	}
}

// String returns the string value of the given option key for the given checkname in this Openstack cloud.
//   - If the option is not set, the value is not changed and false is returned.
//   - If the option is set, the value is set and true is returned.
//   - If the option is set but the value is not a string, an error is returned.
func (opts CloudOptions) String(checkname, key string, value *string) (bool, error) {
	v, found := opts[checkname][key]
	if !found {
		return found, nil
	}

	s, ok := v.(string)
	if !ok {
		return found, fmt.Errorf("%s/%s value is not a string", checkname, key)
	}

	*value = s
	return found, nil
}

// Int returns the int value of the given option key for the given checkname in this Openstack cloud.
//   - If the option is not set, the value is not changed and false is returned.
//   - If the option is set, the value is set and true is returned.
//   - If the option is set but the value is not a string, an error is returned.
func (opts CloudOptions) Int(checkname, key string, value *int) (bool, error) {
	v, found := opts[checkname][key]
	if !found {
		return found, nil
	}

	s, ok := v.(int)
	if !ok {
		return found, fmt.Errorf("%s/%s value is not an int", checkname, key)
	}

	*value = s
	return found, nil
}

// Bool returns the bool value of the given option key for the given checkname in this Openstack cloud.
//   - If the option is not set, the value is not changed and false is returned.
//   - If the option is set, the value is set and true is returned.
//   - If the option is set but the value is not a string, an error is returned.
func (opts CloudOptions) Bool(checkname, key string, value *bool) (bool, error) {
	v, found := opts[checkname][key]
	if !found {
		return found, nil
	}

	s, ok := v.(bool)
	if !ok {
		return found, fmt.Errorf("%s/%s value is not a bool", checkname, key)
	}

	*value = s
	return found, nil
}
