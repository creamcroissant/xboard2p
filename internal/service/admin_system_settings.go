package service

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/async"
	"github.com/creamcroissant/xboard/internal/notifier"
	"github.com/creamcroissant/xboard/internal/repository"
	"github.com/creamcroissant/xboard/internal/security"
)

// AdminSystemSettingsService 负责系统设置读写与密钥/SMTP 测试流程。
type AdminSystemSettingsService interface {
	GetByCategory(ctx context.Context, category string) (map[string]string, error)
	SaveSettings(ctx context.Context, category string, settings map[string]string) error
	Get(ctx context.Context, key string) (string, error)
	TestSMTP(ctx context.Context, config SMTPConfig) error
	ResetCommunicationKey(ctx context.Context) (string, error)
	GetMaskedCommunicationKey(ctx context.Context) (string, error)
	GetCommunicationKey(ctx context.Context) (string, error)
}

// SMTPConfig 描述 SMTP 测试所需字段。
type SMTPConfig struct {
	Host        string
	Port        int
	Encryption  string
	Username    string
	Password    string
	FromAddress string
	ToAddress   string
}

// AdminSystemSettingsOptions 注入系统设置服务依赖。
type AdminSystemSettingsOptions struct {
	Settings          repository.SettingRepository
	NotificationQueue *async.NotificationQueue
	Audit             security.Recorder
	Now               func() time.Time
}

type adminSystemSettingsService struct {
	settings repository.SettingRepository
	queue    *async.NotificationQueue
	audit    security.Recorder
	now      func() time.Time
}

const communicationKeySettingKey = "communication_key" // DEPRECATED: This key is no longer used for Agent authentication (replaced by Per-Agent Token).
const communicationKeyLength = 32
const communicationKeyMaskRune = '*'
const communicationKeyVisibleSuffix = 4

// NewAdminSystemSettingsService 构建系统设置服务。
func NewAdminSystemSettingsService(opts AdminSystemSettingsOptions) AdminSystemSettingsService {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &adminSystemSettingsService{
		settings: opts.Settings,
		queue:    opts.NotificationQueue,
		audit:    opts.Audit,
		now:      nowFn,
	}
}

// GetByCategory 按分类读取设置，返回 key/value 映射。
func (s *adminSystemSettingsService) GetByCategory(ctx context.Context, category string) (map[string]string, error) {
	if s == nil || s.settings == nil {
		return nil, fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	trimmedCategory := strings.TrimSpace(category)
	if trimmedCategory == "" {
		return nil, fmt.Errorf("category is required / 分类不能为空")
	}
	entries, err := s.settings.ListByCategory(ctx, trimmedCategory)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		result[entry.Key] = entry.Value
	}
	return result, nil
}

// SaveSettings 保存分类设置，并做值规范化与缓存失效。
func (s *adminSystemSettingsService) SaveSettings(ctx context.Context, category string, settings map[string]string) error {
	if s == nil || s.settings == nil {
		return fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	trimmedCategory := strings.TrimSpace(category)
	if trimmedCategory == "" {
		return fmt.Errorf("category is required / 分类不能为空")
	}
	if len(settings) == 0 {
		return nil
	}
	now := s.now().Unix()
	for key, value := range settings {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return fmt.Errorf("setting key is required / 设置项 key 不能为空")
		}
		normalized := normalizeSettingValue(trimmedKey, value)
		entry := &repository.Setting{
			Key:       trimmedKey,
			Value:     normalized,
			Category:  trimmedCategory,
			UpdatedAt: now,
		}
		if err := s.settings.Upsert(ctx, entry); err != nil {
			return err
		}
	}
	if s.queue != nil {
		s.queue.InvalidateSettingCache()
	}
	return nil
}

// Get 按 key 获取设置值，未命中返回 ErrNotFound。
func (s *adminSystemSettingsService) Get(ctx context.Context, key string) (string, error) {
	if s == nil || s.settings == nil {
		return "", fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", fmt.Errorf("key is required / key 不能为空")
	}
	entry, err := s.settings.Get(ctx, trimmedKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	if entry == nil {
		return "", ErrNotFound
	}
	return entry.Value, nil
}

// TestSMTP 校验配置并探测连通性，通过后入队发送测试邮件。
func (s *adminSystemSettingsService) TestSMTP(ctx context.Context, config SMTPConfig) error {
	if s == nil {
		return fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	if err := validateSMTPConfig(config); err != nil {
		return err
	}
	if err := ensureSMTPLocalHost(config); err != nil {
		return err
	}
	dialer := smtpDialer{}
	if err := dialer.Test(ctx, config, 10*time.Second); err != nil {
		return err
	}
	if s.queue != nil {
		s.queue.EnqueueEmail(buildSMTPTestEmail(config))
	}
	return nil
}

// ResetCommunicationKey 生成新的通讯密钥并写入 node 分类。
func (s *adminSystemSettingsService) ResetCommunicationKey(ctx context.Context) (string, error) {
	if s == nil || s.settings == nil {
		return "", fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	newKey, err := generateCommunicationKey()
	if err != nil {
		return "", err
	}
	now := s.now().Unix()
	entry := &repository.Setting{
		Key:       communicationKeySettingKey,
		Value:     newKey,
		Category:  "node",
		UpdatedAt: now,
	}
	if err := s.settings.Upsert(ctx, entry); err != nil {
		return "", err
	}
	return newKey, nil
}

// GetMaskedCommunicationKey 返回掩码后的通讯密钥。
func (s *adminSystemSettingsService) GetMaskedCommunicationKey(ctx context.Context) (string, error) {
	key, err := s.loadCommunicationKey(ctx)
	if err != nil {
		return "", err
	}
	return maskSecret(key, communicationKeyMaskRune, communicationKeyVisibleSuffix), nil
}

// GetCommunicationKey 返回明文密钥，并记录审计事件。
func (s *adminSystemSettingsService) GetCommunicationKey(ctx context.Context) (string, error) {
	key, err := s.loadCommunicationKey(ctx)
	if err != nil {
		return "", err
	}
	if s.audit != nil {
		s.audit.Record(ctx, security.Event{
			Kind:     "admin.system.communication_key.reveal",
			ActorID:  "admin",
			Metadata: map[string]any{"setting": communicationKeySettingKey},
		})
	}
	return key, nil
}

// loadCommunicationKey 读取通讯密钥，不存在或为空则生成。
func (s *adminSystemSettingsService) loadCommunicationKey(ctx context.Context) (string, error) {
	if s == nil || s.settings == nil {
		return "", fmt.Errorf("admin system settings service not configured / 系统设置服务未配置")
	}
	entry, err := s.settings.Get(ctx, communicationKeySettingKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return s.ResetCommunicationKey(ctx)
		}
		return "", err
	}
	if entry == nil || strings.TrimSpace(entry.Value) == "" {
		return s.ResetCommunicationKey(ctx)
	}
	return entry.Value, nil
}

// maskSecret 仅保留尾部可见字符，其余使用掩码符。
func maskSecret(value string, maskRune rune, visibleSuffix int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if visibleSuffix < 0 {
		visibleSuffix = 0
	}
	runes := []rune(trimmed)
	length := len(runes)
	if length <= visibleSuffix {
		return string(runes)
	}
	maskedCount := length - visibleSuffix
	masked := strings.Repeat(string(maskRune), maskedCount)
	return masked + string(runes[maskedCount:])
}

// generateCommunicationKey 生成固定长度的随机通讯密钥。
func generateCommunicationKey() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, communicationKeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf), nil
}

// normalizeSettingValue 去除空白并规范化 URL 类配置。
func normalizeSettingValue(key, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if isURLSettingKey(key) {
		return normalizeURLList(trimmed)
	}
	return trimmed
}

// isURLSettingKey 判断 key 是否为 URL 类配置。
func isURLSettingKey(key string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(key)), "url")
}

// normalizeURLList 规范化 URL 列表（去尾斜杠、剔除空项）。
func normalizeURLList(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, ",") {
		parts := strings.Split(trimmed, ",")
		cleaned := make([]string, 0, len(parts))
		for _, part := range parts {
			entry := strings.TrimSpace(part)
			if entry == "" {
				continue
			}
			cleaned = append(cleaned, strings.TrimRight(entry, "/"))
		}
		return strings.Join(cleaned, ",")
	}
	return strings.TrimRight(trimmed, "/")
}

// validateSMTPConfig 校验 SMTP 配置完整性与加密类型。
func validateSMTPConfig(config SMTPConfig) error {
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return fmt.Errorf("SMTP host is required / SMTP 主机不能为空")
	}
	if config.Port <= 0 {
		return fmt.Errorf("SMTP port is required / SMTP 端口不能为空")
	}
	if strings.TrimSpace(config.FromAddress) == "" {
		return fmt.Errorf("SMTP from address is required / SMTP 发件地址不能为空")
	}
	if strings.TrimSpace(config.ToAddress) == "" {
		return fmt.Errorf("SMTP to address is required / SMTP 收件地址不能为空")
	}
	enc := strings.ToLower(strings.TrimSpace(config.Encryption))
	if enc == "" {
		enc = "none"
	}
	switch enc {
	case "none", "starttls", "ssl":
		return nil
	default:
		return fmt.Errorf("unsupported SMTP encryption / 不支持的 SMTP 加密方式")
	}
}

func buildSMTPTestEmail(config SMTPConfig) notifier.EmailRequest {
	subject := "SMTP test"
	if strings.TrimSpace(config.Host) != "" {
		subject = fmt.Sprintf("SMTP test (%s)", strings.TrimSpace(config.Host))
	}
	from := strings.TrimSpace(config.FromAddress)
	to := strings.TrimSpace(config.ToAddress)
	return notifier.EmailRequest{
		To:       to,
		Subject:  subject,
		Template: "smtp_test",
		Body:     "This is a test email from XBoard SMTP settings.",
		Variables: map[string]any{
			"from": from,
			"to":   to,
		},
	}
}

func ensureSMTPLocalHost(config SMTPConfig) error {
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return fmt.Errorf("SMTP host is required / SMTP 主机不能为空")
	}
	if strings.ContainsAny(host, "\r\n\t") {
		return fmt.Errorf("invalid SMTP host / SMTP 主机非法")
	}
	if strings.Contains(host, "://") {
		parsed, err := url.Parse(host)
		if err != nil {
			return fmt.Errorf("invalid SMTP host / SMTP 主机非法")
		}
		host = parsed.Hostname()
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return fmt.Errorf("SMTP host resolve failed / SMTP 主机解析失败")
		}
		for _, candidate := range ips {
			if isPrivateIP(candidate) {
				return fmt.Errorf("SMTP host is private / SMTP 主机为内网地址 (禁止连接内网 SMTP)")
			}
		}
		return nil
	}
	if isPrivateIP(ip) {
		return fmt.Errorf("SMTP host is private / SMTP 主机为内网地址 (禁止连接内网 SMTP)")
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

type smtpDialer struct{}

// Test 尝试建立 SMTP 连接并握手，不发送实际邮件内容。
func (d smtpDialer) Test(ctx context.Context, config SMTPConfig, timeout time.Duration) error {
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return fmt.Errorf("SMTP host is required / SMTP 主机不能为空")
	}
	port := config.Port
	if port <= 0 {
		return fmt.Errorf("SMTP port is required / SMTP 端口不能为空")
	}
	enc := strings.ToLower(strings.TrimSpace(config.Encryption))
	if enc == "" {
		enc = "none"
	}
	address := fmt.Sprintf("%s:%d", host, port)
	dialer := net.Dialer{Timeout: timeout}
	deadline := time.Now().Add(timeout)

	if enc == "ssl" {
		tlsConn, err := tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{ServerName: host})
		if err != nil {
			return fmt.Errorf("SMTP TLS connection failed / SMTP TLS 连接失败: %w", err)
		}
		defer tlsConn.Close()
		_ = tlsConn.SetDeadline(deadline)
		client, err := smtp.NewClient(tlsConn, host)
		if err != nil {
			return fmt.Errorf("SMTP client init failed / SMTP 客户端初始化失败: %w", err)
		}
		defer client.Close()
		if err := client.Hello(host); err != nil {
			return fmt.Errorf("smtp hello failed: %w", err)
		}
		return nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("SMTP connection failed / SMTP 连接失败: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client init failed / SMTP 客户端初始化失败: %w", err)
	}
	defer client.Close()
	if err := client.Hello(host); err != nil {
		return fmt.Errorf("smtp hello failed: %w", err)
	}
	switch enc {
	case "none":
		return nil
	case "starttls":
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP STARTTLS not supported / SMTP 不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("SMTP STARTTLS failed / SMTP STARTTLS 失败: %w", err)
		}
		if err := client.Hello(host); err != nil {
			return fmt.Errorf("smtp hello failed: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported SMTP encryption / 不支持的 SMTP 加密方式")
	}
}
