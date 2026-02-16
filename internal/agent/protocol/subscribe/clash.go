package subscribe

import (
	"regexp"
	"strconv"
	"strings"
)

// ParseClash parses Clash YAML format proxies file.
// The proxies file uses a simplified YAML format with inline JSON-like structures.
func ParseClash(content []byte) ([]ClientConfig, error) {
	lines := strings.Split(string(content), "\n")
	var configs []ClientConfig

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- {") {
			continue
		}

		config := parseClashProxyLine(line)
		if config.Name != "" {
			configs = append(configs, config)
		}
	}

	return configs, nil
}

// parseClashProxyLine parses a single Clash proxy line.
func parseClashProxyLine(line string) ClientConfig {
	config := ClientConfig{}

	// Extract name
	config.Name = extractClashField(line, "name")

	// Extract type
	proxyType := extractClashField(line, "type")
	config.Protocol = proxyType

	// Extract server and port
	config.Server = extractClashField(line, "server")
	if portStr := extractClashField(line, "port"); portStr != "" {
		config.Port, _ = strconv.Atoi(portStr)
	}

	// Extract authentication based on type
	switch proxyType {
	case "vless", "vmess":
		config.UUID = extractClashField(line, "uuid")
		config.Flow = extractClashField(line, "flow")
		config.Network = extractClashField(line, "network")

		// Reality
		if strings.Contains(line, "reality-opts") {
			config.RealityEnabled = true
			config.RealityPublicKey = extractNestedField(line, "reality-opts", "public-key")
			config.RealityShortID = extractNestedField(line, "reality-opts", "short-id")
		}

		// TLS
		if extractClashField(line, "tls") == "true" {
			config.TLS = true
		}
		config.SNI = extractClashField(line, "servername")
		if config.SNI == "" {
			config.SNI = extractClashField(line, "sni")
		}

		// gRPC
		if config.Network == "grpc" {
			config.ServiceName = extractNestedField(line, "grpc-opts", "grpc-service-name")
		}

		// Multiplex
		if strings.Contains(line, "smux") {
			if extractNestedField(line, "smux", "enabled") == "true" {
				config.MuxEnabled = true
			}
			if extractNestedField(line, "smux", "padding") == "true" {
				config.MuxPadding = true
			}
		}

	case "hysteria2":
		config.Password = extractClashField(line, "password")
		config.SNI = extractClashField(line, "sni")
		config.TLS = true // Hysteria2 always uses TLS

		// Port hopping
		config.HopPorts = extractClashField(line, "ports")
		if intervalStr := extractClashField(line, "HopInterval"); intervalStr != "" {
			config.HopInterval, _ = strconv.Atoi(intervalStr)
		}

		// Bandwidth
		config.UpMbps = parseClashBandwidth(extractClashField(line, "up"))
		config.DownMbps = parseClashBandwidth(extractClashField(line, "down"))

		// Fingerprint
		config.Fingerprint = extractClashField(line, "fingerprint")

	case "tuic":
		config.UUID = extractClashField(line, "uuid")
		config.Password = extractClashField(line, "password")
		config.SNI = extractClashField(line, "sni")
		config.TLS = true // TUIC always uses TLS
		config.CongestionControl = extractClashField(line, "congestion-controller")
		config.Fingerprint = extractClashField(line, "fingerprint")

	case "ss":
		config.Protocol = "shadowsocks"
		config.Cipher = extractClashField(line, "cipher")
		config.Password = extractClashField(line, "password")

		// Check for ShadowTLS plugin
		if extractClashField(line, "plugin") == "shadow-tls" {
			config.ShadowTLSPassword = extractNestedField(line, "plugin-opts", "password")
			if versionStr := extractNestedField(line, "plugin-opts", "version"); versionStr != "" {
				config.ShadowTLSVersion, _ = strconv.Atoi(versionStr)
			}
			config.SNI = extractNestedField(line, "plugin-opts", "host")
		}

		// Multiplex
		if strings.Contains(line, "smux") {
			if extractNestedField(line, "smux", "enabled") == "true" {
				config.MuxEnabled = true
			}
			if extractNestedField(line, "smux", "padding") == "true" {
				config.MuxPadding = true
			}
		}

	case "trojan":
		config.Password = extractClashField(line, "password")
		config.SNI = extractClashField(line, "sni")
		config.TLS = true // Trojan always uses TLS
		config.Fingerprint = extractClashField(line, "fingerprint")

		// Multiplex
		if strings.Contains(line, "smux") {
			if extractNestedField(line, "smux", "enabled") == "true" {
				config.MuxEnabled = true
			}
		}

	case "anytls":
		config.Password = extractClashField(line, "password")
		config.SNI = extractClashField(line, "sni")
		config.TLS = true
		config.Fingerprint = extractClashField(line, "fingerprint")
	}

	// Skip cert verify
	if extractClashField(line, "skip-cert-verify") == "true" {
		config.Insecure = true
	}

	return config
}

// extractClashField extracts a field value from Clash proxy line.
func extractClashField(line, field string) string {
	// Pattern: field: value or field: "value"
	patterns := []string{
		field + `: "([^"]*)"`,           // quoted value
		field + `: '([^']*)'`,           // single quoted value
		field + `: ([^,}\s]+)`,          // unquoted value
		`"` + field + `": "([^"]*)"`,    // JSON style quoted
		`"` + field + `": ([^,}\s]+)`,   // JSON style unquoted
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

// extractNestedField extracts a field from a nested structure like "opts: { field: value }".
func extractNestedField(line, parent, field string) string {
	// Find the parent structure
	parentPattern := regexp.MustCompile(parent + `:\s*\{([^}]+)\}`)
	parentMatch := parentPattern.FindStringSubmatch(line)
	if len(parentMatch) < 2 {
		return ""
	}

	// Extract field from within the parent
	return extractClashField(parentMatch[1], field)
}

// parseClashBandwidth parses bandwidth string like "200 Mbps" to int.
func parseClashBandwidth(s string) int {
	if s == "" {
		return 0
	}
	// Remove "Mbps" suffix and parse
	s = strings.TrimSuffix(s, " Mbps")
	s = strings.TrimSuffix(s, "Mbps")
	val, _ := strconv.Atoi(strings.TrimSpace(s))
	return val
}

// extractClashProxy extracts a specific proxy entry from Clash proxies content.
func extractClashProxy(content, name string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, `name: "`+name+`"`) || strings.Contains(line, `name: '`+name+`'`) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
