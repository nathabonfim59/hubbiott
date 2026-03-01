// Package embed provides embedded template files for hubbiott.
// These templates are used for bootstrapping configuration and systemd services.
package embed

import "embed"

// FS contains embedded template files.
//
//go:embed config.yaml.tmpl hubbiott.service.tmpl install.sh
var FS embed.FS

// ConfigTemplate is the raw content of the config.yaml.tmpl file.
//
//go:embed config.yaml.tmpl
var ConfigTemplate string

// SystemdServiceTemplate is the raw content of the hubbiott.service.tmpl file.
//
//go:embed hubbiott.service.tmpl
var SystemdServiceTemplate string

// InstallScript is the raw content of the install.sh file.
//
//go:embed install.sh
var InstallScript string
