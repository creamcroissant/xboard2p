// Package protocol 提供 sing-box 协议配置管理能力。
package protocol

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/initsys"
	"github.com/creamcroissant/xboard/internal/agent/protocol/parser"
)

// Config 定义协议管理器的配置。
type Config struct {
	// ConfigDir 是 sing-box 配置文件所在目录
	ConfigDir string `yaml:"config_dir"`

	// ServiceName 是 sing-box 服务名称
	ServiceName string `yaml:"service_name"`

	// ValidateCmd 是应用配置前执行的校验命令
	ValidateCmd string `yaml:"validate_cmd"`

	// AutoRestart 控制配置变更后是否重启服务
	AutoRestart bool `yaml:"auto_restart"`

	// PreHook 是应用配置前执行的钩子命令
	PreHook string `yaml:"pre_hook"`

	// PostHook 是应用配置后执行的钩子命令
	PostHook string `yaml:"post_hook"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		ConfigDir:   "/etc/sing-box/conf",
		ServiceName: "sing-box",
		ValidateCmd: "",
		AutoRestart: true,
	}
}

// ConfigFile 描述协议配置文件元信息。
type ConfigFile struct {
	Filename    string    `json:"filename"`
	ModTime     time.Time `json:"mod_time"`
	Size        int64     `json:"size"`
	ContentHash string    `json:"content_hash"`
	Content     []byte    `json:"content,omitempty"`
}

// Manager 管理 sing-box 协议配置。
type Manager struct {
	cfg      Config
	init     initsys.InitSystem
	registry *parser.Registry
}

// NewManager 创建协议管理器实例。
func NewManager(cfg Config, initSys initsys.InitSystem) *Manager {
	if initSys == nil {
		initSys = initsys.Detect()
	}
	return &Manager{
		cfg:      cfg,
		init:     initSys,
		registry: parser.NewRegistry(),
	}
}

// InitSystemType 返回当前使用的 init 系统类型。
func (m *Manager) InitSystemType() string {
	return m.init.Type()
}

// ListConfigs 返回所有协议配置文件。
func (m *Manager) ListConfigs() ([]ConfigFile, error) {
	pattern := filepath.Join(m.cfg.ConfigDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob config files: %w", err)
	}

	configs := make([]ConfigFile, 0, len(files))
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		hash := md5.Sum(content)
		configs = append(configs, ConfigFile{
			Filename:    filepath.Base(f),
			ModTime:     info.ModTime(),
			Size:        info.Size(),
			ContentHash: hex.EncodeToString(hash[:]),
		})
	}

	// 按文件名排序
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Filename < configs[j].Filename
	})

	return configs, nil
}

// ListInbounds 仅返回 inbound 相关配置文件。
func (m *Manager) ListInbounds() ([]ConfigFile, error) {
	configs, err := m.ListConfigs()
	if err != nil {
		return nil, err
	}

	inbounds := make([]ConfigFile, 0)
	for _, c := range configs {
		if strings.Contains(c.Filename, "inbounds") {
			inbounds = append(inbounds, c)
		}
	}
	return inbounds, nil
}

// ReadConfig 读取指定配置文件内容。
func (m *Manager) ReadConfig(filename string) ([]byte, error) {
	name, err := sanitizeFilename(filename)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(m.cfg.ConfigDir, name)
	return os.ReadFile(path)
}

// WriteConfig 写入配置文件。
func (m *Manager) WriteConfig(filename string, content []byte) error {
	name, err := sanitizeFilename(filename)
	if err != nil {
		return err
	}
	// 确保配置目录存在
	if err := os.MkdirAll(m.cfg.ConfigDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(m.cfg.ConfigDir, name)

	// 校验 JSON 格式
	if !json.Valid(content) {
		return fmt.Errorf("invalid JSON content")
	}

	// 美化 JSON 输出
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, content, "", "  "); err != nil {
		return fmt.Errorf("format JSON: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(path, prettyJSON.Bytes(), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// DeleteConfig 删除配置文件。
func (m *Manager) DeleteConfig(filename string) error {
	name, err := sanitizeFilename(filename)
	if err != nil {
		return err
	}
	path := filepath.Join(m.cfg.ConfigDir, name)
	return os.Remove(path)
}

// CreateFromTemplate 根据模板生成配置文件。
func (m *Manager) CreateFromTemplate(filename, tmplContent string, vars map[string]any) error {
	// 解析模板
	tmpl, err := template.New("config").Funcs(templateFuncs()).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// 执行模板渲染
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return m.WriteConfig(filename, buf.Bytes())
}

// ValidateConfig 校验当前配置（若未配置校验命令则跳过）。
func (m *Manager) ValidateConfig(ctx context.Context) error {
	if m.cfg.ValidateCmd == "" {
		return nil
	}

	cmd := strings.ReplaceAll(m.cfg.ValidateCmd, "{config_dir}", m.cfg.ConfigDir)
	return runCommand(ctx, cmd)
}

// ReloadService 重载或重启 sing-box 服务。
func (m *Manager) ReloadService(ctx context.Context) error {
	// 执行预处理钩子
	if m.cfg.PreHook != "" {
		if err := runCommand(ctx, m.cfg.PreHook); err != nil {
			return fmt.Errorf("pre-hook failed: %w", err)
		}
	}

	// 校验配置（如启用校验命令）
	if err := m.ValidateConfig(ctx); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 按需重启服务
	if m.cfg.AutoRestart {
		if err := m.init.Restart(ctx, m.cfg.ServiceName); err != nil {
			return fmt.Errorf("restart service: %w", err)
		}
	}

	// 执行后置钩子
	if m.cfg.PostHook != "" {
		if err := runCommand(ctx, m.cfg.PostHook); err != nil {
			return fmt.Errorf("post-hook failed: %w", err)
		}
	}

	return nil
}

// ServiceStatus 返回 sing-box 服务是否运行中。
func (m *Manager) ServiceStatus(ctx context.Context) (bool, error) {
	return m.init.Status(ctx, m.cfg.ServiceName)
}

// ApplyConfig 写入主配置并重载服务。
func (m *Manager) ApplyConfig(ctx context.Context, content []byte) error {
	// 默认写入主配置文件，暂不支持多配置合并
	const defaultFilename = "config.json"

	if err := m.WriteConfig(defaultFilename, content); err != nil {
		return err
	}
	return m.ReloadService(ctx)
}

// ApplyFromTemplate 根据模板生成配置并重载服务。
func (m *Manager) ApplyFromTemplate(ctx context.Context, filename, tmpl string, vars map[string]any) error {
	if err := m.CreateFromTemplate(filename, tmpl, vars); err != nil {
		return err
	}
	return m.ReloadService(ctx)
}

// RemoveConfig 删除配置并重载服务。
func (m *Manager) RemoveConfig(ctx context.Context, filename string) error {
	if err := m.DeleteConfig(filename); err != nil {
		return err
	}
	return m.ReloadService(ctx)
}

// templateFuncs 返回模板函数集合。
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"default": func(def, val any) any {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"json": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		},
	}
}

func sanitizeFilename(filename string) (string, error) {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return "", fmt.Errorf("filename is required")
	}
	if strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("invalid filename")
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", fmt.Errorf("invalid filename")
	}
	base := filepath.Base(cleaned)
	if base != cleaned {
		return "", fmt.Errorf("invalid filename")
	}
	return base, nil
}

// ListConfigsWithDetails 返回带解析详情的配置文件列表。
func (m *Manager) ListConfigsWithDetails() ([]parser.ConfigFileWithDetails, error) {
	pattern := filepath.Join(m.cfg.ConfigDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob config files: %w", err)
	}

	results := make([]parser.ConfigFileWithDetails, 0, len(files))
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		hash := md5.Sum(content)
		filename := filepath.Base(f)

		// 解析配置文件并提取协议
		protocols := m.registry.ParseAll(filename, content)

		// 仅保留包含 inbound 协议的文件
		// 跳过 log/route/dns/ntp 等非协议文件
		if len(protocols) == 0 {
			continue
		}

		results = append(results, parser.ConfigFileWithDetails{
			Filename:    filename,
			ContentHash: hex.EncodeToString(hash[:]),
			Protocols:   protocols,
		})
	}

	// 按文件名排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Filename < results[j].Filename
	})

	return results, nil
}

// GetRegistry 返回协议解析注册表。
func (m *Manager) GetRegistry() *parser.Registry {
	return m.registry
}

// UserConfig 描述注入到协议配置中的用户信息。
type UserConfig struct {
	UUID    string
	Email   string
	Enabled bool
}

// InjectUsers 将用户注入配置并重载服务。
// 流程：读取配置 -> 更新 inbounds -> 写回 -> 重载
func (m *Manager) InjectUsers(ctx context.Context, users []UserConfig) error {
	// 读取当前配置文件
	const defaultFilename = "config.json"
	content, err := m.ReadConfig(defaultFilename)
	if err != nil {
		return fmt.Errorf("read config for user injection: %w", err)
	}

	// 将配置解析为通用 JSON 对象
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parse config JSON: %w", err)
	}

	// 更新 inbounds 中的用户
	if err := m.updateInboundUsers(config, users); err != nil {
		return fmt.Errorf("update inbound users: %w", err)
	}

	// 重新序列化为 JSON
	updatedContent, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal updated config: %w", err)
	}

	// 写回配置并重载服务
	if err := m.WriteConfig(defaultFilename, updatedContent); err != nil {
		return fmt.Errorf("write updated config: %w", err)
	}

	return m.ReloadService(ctx)
}

// updateInboundUsers 更新配置中的所有 inbound 用户。
func (m *Manager) updateInboundUsers(config map[string]any, users []UserConfig) error {
	// 查找 "inbounds" 数组（Sing-box 格式）
	if inbounds, ok := config["inbounds"].([]any); ok {
		for _, inbound := range inbounds {
			if inboundMap, ok := inbound.(map[string]any); ok {
				m.injectUsersIntoSingboxInbound(inboundMap, users)
			}
		}
		return nil
	}

	// 预留 Xray 格式兼容（inbounds + settings.clients 结构）
	return nil
}

// injectUsersIntoSingboxInbound 注入 sing-box inbound 用户。
func (m *Manager) injectUsersIntoSingboxInbound(inbound map[string]any, users []UserConfig) {
	inboundType, _ := inbound["type"].(string)

	// 按协议类型构建用户列表
	usersList := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if !u.Enabled {
			continue
		}
		user := map[string]any{
			"name": u.Email,
		}
		switch inboundType {
		case "vless":
			user["uuid"] = u.UUID
			user["flow"] = "xtls-rprx-vision"
		case "vmess":
			user["uuid"] = u.UUID
		case "shadowsocks":
			user["password"] = u.UUID
		case "trojan", "hysteria2", "tuic":
			user["password"] = u.UUID
		default:
			// 未知协议，跳过注入但记录日志
			slog.Warn("unknown inbound type, skipping user injection", "type", inboundType)
			return
		}
		usersList = append(usersList, user)
	}

	if len(usersList) > 0 {
		inbound["users"] = usersList
	}
}

// InjectUsersXray 将用户注入 Xray 配置并重载服务。
func (m *Manager) InjectUsersXray(ctx context.Context, users []UserConfig) error {
	// 读取当前配置文件
	const defaultFilename = "config.json"
	content, err := m.ReadConfig(defaultFilename)
	if err != nil {
		return fmt.Errorf("read config for user injection: %w", err)
	}

	// 将配置解析为通用 JSON 对象
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parse config JSON: %w", err)
	}

	// 更新 Xray inbounds 中的用户
	if err := m.updateXrayInboundUsers(config, users); err != nil {
		return fmt.Errorf("update xray inbound users: %w", err)
	}

	// 重新序列化为 JSON
	updatedContent, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal updated config: %w", err)
	}

	// 写回配置并重载服务
	if err := m.WriteConfig(defaultFilename, updatedContent); err != nil {
		return fmt.Errorf("write updated config: %w", err)
	}

	return m.ReloadService(ctx)
}

// updateXrayInboundUsers 更新 Xray 配置中的 inbound 用户。
func (m *Manager) updateXrayInboundUsers(config map[string]any, users []UserConfig) error {
	inbounds, ok := config["inbounds"].([]any)
	if !ok {
		return nil
	}

	for _, inbound := range inbounds {
		inboundMap, ok := inbound.(map[string]any)
		if !ok {
			continue
		}
		m.injectUsersIntoXrayInbound(inboundMap, users)
	}

	return nil
}

// injectUsersIntoXrayInbound 注入 Xray inbound 用户。
func (m *Manager) injectUsersIntoXrayInbound(inbound map[string]any, users []UserConfig) {
	protocol, _ := inbound["protocol"].(string)

	// 获取或创建 settings 对象
	settings, ok := inbound["settings"].(map[string]any)
	if !ok {
		settings = make(map[string]any)
		inbound["settings"] = settings
	}

	// 按协议类型构建 clients 列表
	clients := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if !u.Enabled {
			continue
		}
		client := map[string]any{
			"email": u.Email,
			"level": 0,
		}
		switch protocol {
		case "vless":
			client["id"] = u.UUID
			client["flow"] = "xtls-rprx-vision"
		case "vmess":
			client["id"] = u.UUID
			client["alterId"] = 0
		case "shadowsocks":
			client["password"] = u.UUID
		case "trojan":
			client["password"] = u.UUID
		default:
			// 未知协议，跳过注入但记录日志
			slog.Warn("unknown inbound protocol, skipping user injection", "protocol", protocol)
			return
		}
		clients = append(clients, client)
	}

	if len(clients) > 0 {
		settings["clients"] = clients
	}
}

// DetectCoreType 尝试判断配置属于 sing-box 还是 xray。
func (m *Manager) DetectCoreType() string {
	const defaultFilename = "config.json"
	content, err := m.ReadConfig(defaultFilename)
	if err != nil {
		return "unknown"
	}

	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		return "unknown"
	}

	// 检查 Xray 特有字段
	if _, hasAPI := config["api"]; hasAPI {
		return "xray"
	}
	if _, hasPolicy := config["policy"]; hasPolicy {
		return "xray"
	}

	// 检查 Sing-box 特有字段
	if _, hasExperimental := config["experimental"]; hasExperimental {
		return "sing-box"
	}

	// 检查 inbound 结构
	if inbounds, ok := config["inbounds"].([]any); ok && len(inbounds) > 0 {
		if inbound, ok := inbounds[0].(map[string]any); ok {
			// Xray 使用 "protocol" 字段，Sing-box 使用 "type" 字段
			if _, hasProtocol := inbound["protocol"]; hasProtocol {
				return "xray"
			}
			if _, hasType := inbound["type"]; hasType {
				return "sing-box"
			}
		}
	}

	return "unknown"
}
