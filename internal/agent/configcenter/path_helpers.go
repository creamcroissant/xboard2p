package configcenter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creamcroissant/xboard/internal/agent/config"
	"github.com/creamcroissant/xboard/internal/agent/protocol"
)

func resolveProtocolPaths(cfg config.ProtocolConfig) (legacyDirAbs, managedDirAbs, mergeOutputFile string, err error) {
	legacyDir := strings.TrimSpace(cfg.LegacyConfigDir)
	managedDir := strings.TrimSpace(cfg.ManagedConfigDir)
	if managedDir == "" {
		managedDir = strings.TrimSpace(cfg.ConfigDir)
	}
	if legacyDir == "" {
		legacyDir = strings.TrimSpace(cfg.ConfigDir)
	}
	if managedDir == "" || legacyDir == "" {
		return "", "", "", fmt.Errorf("managed/legacy config directories are required")
	}

	managedDirAbs, err = filepath.Abs(filepath.Clean(managedDir))
	if err != nil {
		return "", "", "", fmt.Errorf("resolve managed config dir: %w", err)
	}
	legacyDirAbs, err = filepath.Abs(filepath.Clean(legacyDir))
	if err != nil {
		return "", "", "", fmt.Errorf("resolve legacy config dir: %w", err)
	}
	if managedDirAbs == string(filepath.Separator) || legacyDirAbs == string(filepath.Separator) {
		return "", "", "", fmt.Errorf("managed/legacy config directory must not be root path")
	}

	mergeOutputFile = strings.TrimSpace(cfg.MergeOutputFile)
	if mergeOutputFile == "" {
		mergeOutputFile = "config.json"
	}
	mergeOutputFile, err = sanitizeFilename(mergeOutputFile)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid merge output file: %w", err)
	}

	return legacyDirAbs, managedDirAbs, mergeOutputFile, nil
}

func sanitizeFilename(filename string) (string, error) {
	return protocol.SanitizeFilename(filename)
}

func normalizeCoreType(coreType string) string {
	return protocol.NormalizeCoreType(coreType)
}
