package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/core"
)

const defaultDrainTimeout = 5 * time.Second

// SwitcherConfig controls zero-downtime switching behavior.
type SwitcherConfig struct {
	PortRangeStart int
	PortRangeEnd   int
	MaxRetries     int
	HealthTimeout  time.Duration
	HealthInterval time.Duration
	DrainTimeout   time.Duration
	NftBin         string
	ConntrackBin   string
	NftTableName   string
	PIDDir         string
	CgroupBasePath string
}

// Dependency interfaces for Switcher.
type dnatApplier interface {
	EnsureInfrastructure(ctx context.Context) error
	SwitchAtomic(ctx context.Context, rules []*DNATRule) error
}

type portAllocator interface {
	AllocateWithRetry(ctx context.Context, externalPorts []int, startFn func(map[int]int) error, maxRetries int) (map[int]int, error)
}

type conntrackFlusher interface {
	FlushAllProtocols(ctx context.Context, port int) error
}

type configPatcher interface {
	Patch(coreType string, configJSON []byte, portMappings map[int]int) ([]byte, error)
}

type stateRebuilder interface {
	RebuildState(ctx context.Context) ([]PortMapping, error)
	GetOccupiedInternalPorts(ctx context.Context) (map[int]bool, error)
}

type healthChecker interface {
	CheckPorts(ctx context.Context, ports []int) error
}

type groupLock interface {
	TryLock(groupID string) bool
	Unlock(groupID string)
}

type orphanCleaner interface {
	WritePIDFile(instanceID string, pid int, coreType string, ports []int) error
	MarkDraining(instanceID string) error
	RemovePIDFile(instanceID string) error
	CleanupOrphans(ctx context.Context) error
}

type cgroupManager interface {
	IsSupported() bool
	AddProcess(group string, pid int) error
	KillGroup(group string) error
}

// SwitchRequest describes a switch operation.
type SwitchRequest struct {
	FromInstanceID string
	ToCoreType     string
	ConfigJSON     []byte
	ListenPorts    []int
}

// SwitchResult reports switch outcome.
type SwitchResult struct {
	Success       bool
	NewInstanceID string
	PortMappings  map[int]int
	Error         string
}

// InstanceGroup tracks a port group state.
type InstanceGroup struct {
	ID            string
	ExternalPorts []int
	InternalPorts []int
	CoreType      string
	InstanceID    string
}

// SwitcherOptions configures Switcher dependencies.
type SwitcherOptions struct {
	CoreManager    *core.Manager
	OutputPath     string
	Logger         *slog.Logger
	Config         SwitcherConfig
	DNATManager    dnatApplier
	PortAllocator  portAllocator
	Conntrack      conntrackFlusher
	ConfigPatcher  configPatcher
	StateRebuilder stateRebuilder
	HealthChecker  healthChecker
	GroupLocks     groupLock
	OrphanCleaner  orphanCleaner
	CgroupManager  cgroupManager
}

// Switcher coordinates zero-downtime switching.
type Switcher struct {
	coreMgr        *core.Manager
	dnatMgr        dnatApplier
	portAlloc      portAllocator
	conntrack      conntrackFlusher
	configPatcher  configPatcher
	stateRebuilder stateRebuilder
	healthChecker  healthChecker
	groupLocks     groupLock
	orphanCleaner  orphanCleaner
	cgroupMgr      cgroupManager
	outputPath     string
	config         SwitcherConfig
	logger         *slog.Logger

	groups   map[string]*InstanceGroup
	groupsMu sync.RWMutex

	nftApplyMu sync.Mutex
}

// NewSwitcher creates a Switcher with default dependencies where omitted.
func NewSwitcher(opts SwitcherOptions) (*Switcher, error) {
	if opts.CoreManager == nil {
		return nil, fmt.Errorf("core manager is required")
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	cfg := normalizeSwitcherConfig(opts.Config)

	s := &Switcher{
		coreMgr:    opts.CoreManager,
		outputPath: opts.OutputPath,
		config:     cfg,
		logger:     logger,
		groups:     make(map[string]*InstanceGroup),
	}

	if opts.DNATManager != nil {
		s.dnatMgr = opts.DNATManager
	} else {
		s.dnatMgr = NewDNATManager(cfg.NftBin, cfg.NftTableName, logger)
	}
	if opts.PortAllocator != nil {
		s.portAlloc = opts.PortAllocator
	} else {
		s.portAlloc = NewPortAllocator(cfg.PortRangeStart, cfg.PortRangeEnd)
	}
	if opts.Conntrack != nil {
		s.conntrack = opts.Conntrack
	} else {
		s.conntrack = NewConntrackFlusher(cfg.ConntrackBin, logger)
	}
	if opts.ConfigPatcher != nil {
		s.configPatcher = opts.ConfigPatcher
	} else {
		s.configPatcher = NewConfigPatcher(logger)
	}
	if opts.StateRebuilder != nil {
		s.stateRebuilder = opts.StateRebuilder
	} else {
		s.stateRebuilder = NewStateRebuilder(cfg.NftBin, cfg.NftTableName, logger)
	}
	if opts.HealthChecker != nil {
		s.healthChecker = opts.HealthChecker
	} else {
		s.healthChecker = NewHealthChecker(cfg.HealthTimeout, cfg.HealthInterval)
	}
	if opts.GroupLocks != nil {
		s.groupLocks = opts.GroupLocks
	} else {
		s.groupLocks = NewGroupLockManager()
	}
	if opts.OrphanCleaner != nil {
		s.orphanCleaner = opts.OrphanCleaner
	} else {
		s.orphanCleaner = NewOrphanCleaner(cfg.PIDDir, logger)
	}
	if opts.CgroupManager != nil {
		s.cgroupMgr = opts.CgroupManager
	} else {
		s.cgroupMgr = NewCgroupManager(cfg.CgroupBasePath, logger)
	}

	return s, nil
}

// Initialize prepares nftables state and performs orphan cleanup.
func (s *Switcher) Initialize(ctx context.Context) error {
	if s.dnatMgr != nil {
		if err := s.dnatMgr.EnsureInfrastructure(ctx); err != nil {
			return err
		}
	}
	if s.orphanCleaner != nil {
		if err := s.orphanCleaner.CleanupOrphans(ctx); err != nil {
			s.logger.Warn("cleanup orphans failed", "error", err)
		}
	}
	return nil
}

// Switch executes a zero-downtime switch.
func (s *Switcher) Switch(ctx context.Context, req SwitchRequest) (*SwitchResult, error) {
	if req.ToCoreType == "" {
		return nil, fmt.Errorf("to_core_type is required")
	}
	if len(req.ConfigJSON) == 0 {
		return nil, fmt.Errorf("config_json is required")
	}
	if !json.Valid(req.ConfigJSON) {
		return nil, fmt.Errorf("config_json is not valid JSON")
	}
	if len(req.ListenPorts) == 0 {
		return nil, fmt.Errorf("listen ports are required")
	}

	groupID := ComputeGroupID(req.ListenPorts)
	if groupID == "" {
		return nil, fmt.Errorf("invalid group id")
	}
	if !s.groupLocks.TryLock(groupID) {
		return nil, fmt.Errorf("switch for group %s already in progress", groupID)
	}
	defer s.groupLocks.Unlock(groupID)

	occupied, err := s.stateRebuilder.GetOccupiedInternalPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("rebuild occupied ports: %w", err)
	}

	var (
		newInstanceID string
		internalPorts []int
		portMappings  map[int]int
	)

	startFn := func(mappings map[int]int) error {
		ports := internalPortsFromMappings(req.ListenPorts, mappings)
		if hasMappingConflict(occupied, ports) {
			return fmt.Errorf("address already in use")
		}

		patched, err := s.configPatcher.Patch(req.ToCoreType, req.ConfigJSON, mappings)
		if err != nil {
			return err
		}

		path, err := s.writeConfig(groupID, patched)
		if err != nil {
			return err
		}

		instanceID := fmt.Sprintf("%s-%d", req.ToCoreType, time.Now().UnixNano())
		if err := s.coreMgr.StartInstance(ctx, core.CoreType(req.ToCoreType), instanceID, path, ports); err != nil {
			return err
		}

		newInstanceID = instanceID
		internalPorts = ports
		portMappings = mappings
		return nil
	}

	_, err = s.portAlloc.AllocateWithRetry(ctx, req.ListenPorts, startFn, s.config.MaxRetries)
	if err != nil {
		return nil, err
	}

	pidFileWritten := false
	if err := s.ensurePIDTracking(ctx, req.ToCoreType, newInstanceID, internalPorts); err != nil {
		s.cleanupNewInstance(newInstanceID, pidFileWritten)
		return nil, err
	}
	pidFileWritten = true

	if err := s.healthChecker.CheckPorts(ctx, internalPorts); err != nil {
		s.cleanupNewInstance(newInstanceID, pidFileWritten)
		return nil, err
	}

	s.nftApplyMu.Lock()
	rules, err := s.buildRules(ctx, portMappings)
	if err != nil {
		s.nftApplyMu.Unlock()
		s.cleanupNewInstance(newInstanceID, pidFileWritten)
		return nil, err
	}

	if err := s.dnatMgr.SwitchAtomic(ctx, rules); err != nil {
		s.nftApplyMu.Unlock()
		s.cleanupNewInstance(newInstanceID, pidFileWritten)
		return nil, err
	}
	s.nftApplyMu.Unlock()

	for _, port := range req.ListenPorts {
		if err := s.conntrack.FlushAllProtocols(ctx, port); err != nil {
			s.logger.Warn("conntrack flush failed", "port", port, "error", err)
		}
	}

	s.setGroup(&InstanceGroup{
		ID:            groupID,
		ExternalPorts: clonePorts(req.ListenPorts),
		InternalPorts: clonePorts(internalPorts),
		CoreType:      req.ToCoreType,
		InstanceID:    newInstanceID,
	})

	if req.FromInstanceID != "" && req.FromInstanceID != newInstanceID {
		s.asyncCleanup(req.FromInstanceID)
	}

	return &SwitchResult{
		Success:       true,
		NewInstanceID: newInstanceID,
		PortMappings:  portMappings,
	}, nil
}

// GetGroup returns a snapshot of the group by id.
func (s *Switcher) GetGroup(groupID string) *InstanceGroup {
	s.groupsMu.RLock()
	defer s.groupsMu.RUnlock()
	group := s.groups[groupID]
	if group == nil {
		return nil
	}
	return cloneGroup(group)
}

// ListGroups returns all groups tracked by Switcher.
func (s *Switcher) ListGroups() []*InstanceGroup {
	s.groupsMu.RLock()
	defer s.groupsMu.RUnlock()

	groups := make([]*InstanceGroup, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, cloneGroup(group))
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	return groups
}

// Shutdown performs best-effort cleanup.
func (s *Switcher) Shutdown(ctx context.Context) error {
	if s.orphanCleaner != nil {
		if err := s.orphanCleaner.CleanupOrphans(ctx); err != nil {
			s.logger.Warn("cleanup orphans failed", "error", err)
		}
	}
	return nil
}

func (s *Switcher) setGroup(group *InstanceGroup) {
	if group == nil {
		return
	}
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()
	s.groups[group.ID] = cloneGroup(group)
}

func (s *Switcher) asyncCleanup(instanceID string) {
	if instanceID == "" {
		return
	}
	go func() {
		if s.orphanCleaner != nil {
			if err := s.orphanCleaner.MarkDraining(instanceID); err != nil {
				s.logger.Warn("mark draining failed", "instance_id", instanceID, "error", err)
			}
		}

		time.Sleep(s.config.DrainTimeout)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stopErr := s.coreMgr.StopInstance(ctx, instanceID)
		if stopErr != nil {
			s.logger.Warn("stop old instance failed", "instance_id", instanceID, "error", stopErr)
			if s.cgroupMgr != nil && s.cgroupMgr.IsSupported() {
				if err := s.cgroupMgr.KillGroup(instanceID); err != nil {
					s.logger.Warn("kill cgroup failed", "instance_id", instanceID, "error", err)
				}
			}
		}

		if s.orphanCleaner != nil {
			if err := s.orphanCleaner.RemovePIDFile(instanceID); err != nil {
				s.logger.Warn("remove pid file failed", "instance_id", instanceID, "error", err)
			}
		}
	}()
}

func (s *Switcher) ensurePIDTracking(ctx context.Context, coreType string, instanceID string, ports []int) error {
	if s.orphanCleaner == nil {
		return nil
	}
	coreImpl, ok := s.coreMgr.GetCore(core.CoreType(coreType))
	if !ok {
		return fmt.Errorf("core not registered: %s", coreType)
	}

	inst, err := coreImpl.Status(ctx, instanceID)
	if err != nil {
		return err
	}

	if inst == nil {
		s.logger.Warn("instance status is nil", "instance_id", instanceID)
		return nil
	}

	if inst.PID <= 0 {
		s.logger.Warn("instance pid not available, skip pid tracking", "instance_id", instanceID)
		return nil
	}

	if s.cgroupMgr != nil && s.cgroupMgr.IsSupported() {
		if err := s.cgroupMgr.AddProcess(instanceID, inst.PID); err != nil {
			s.logger.Warn("add process to cgroup failed", "instance_id", instanceID, "error", err)
		}
	}
	return s.orphanCleaner.WritePIDFile(instanceID, inst.PID, string(coreType), ports)
}

func (s *Switcher) cleanupNewInstance(instanceID string, pidFileWritten bool) {
	if instanceID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.coreMgr.StopInstance(ctx, instanceID); err != nil {
		s.logger.Warn("stop instance failed", "instance_id", instanceID, "error", err)
		if s.cgroupMgr != nil && s.cgroupMgr.IsSupported() {
			if err := s.cgroupMgr.KillGroup(instanceID); err != nil {
				s.logger.Warn("kill cgroup failed", "instance_id", instanceID, "error", err)
			}
		}
	}
	if pidFileWritten && s.orphanCleaner != nil {
		if err := s.orphanCleaner.RemovePIDFile(instanceID); err != nil {
			s.logger.Warn("remove pid file failed", "instance_id", instanceID, "error", err)
		}
	}
}

func (s *Switcher) writeConfig(prefix string, content []byte) (string, error) {
	baseDir := "."
	if strings.TrimSpace(s.outputPath) != "" {
		baseDir = filepath.Dir(s.outputPath)
	}
	if baseDir == "" {
		baseDir = "."
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	name := "core-switch-" + sanitizeToken(prefix)
	if name == "core-switch-" {
		name = fmt.Sprintf("core-switch-%d", time.Now().UnixNano())
	}
	path := filepath.Join(baseDir, name+".json")
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}

func (s *Switcher) buildRules(ctx context.Context, overrides map[int]int) ([]*DNATRule, error) {
	mappings, err := s.stateRebuilder.RebuildState(ctx)
	if err != nil {
		return nil, err
	}

	type ruleInfo struct {
		internal int
		tcp      bool
		udp      bool
	}

	byPort := make(map[int]*ruleInfo)
	for _, mapping := range mappings {
		info := byPort[mapping.ExternalPort]
		if info == nil {
			info = &ruleInfo{internal: mapping.InternalPort}
			byPort[mapping.ExternalPort] = info
		} else if mapping.InternalPort != 0 {
			info.internal = mapping.InternalPort
		}
		protocol := strings.ToLower(strings.TrimSpace(mapping.Protocol))
		switch protocol {
		case "udp":
			info.udp = true
		default:
			info.tcp = true
		}
	}

	for external, internal := range overrides {
		info := byPort[external]
		if info == nil {
			info = &ruleInfo{}
			byPort[external] = info
		}
		info.internal = internal
		if !info.tcp && !info.udp {
			info.tcp = true
			info.udp = true
		}
	}

	rules := make([]*DNATRule, 0, len(byPort))
	for external, info := range byPort {
		if external <= 0 || info == nil || info.internal <= 0 {
			continue
		}
		protocol := "tcp"
		if info.tcp && info.udp {
			protocol = "both"
		} else if info.udp {
			protocol = "udp"
		}
		rules = append(rules, &DNATRule{ExternalPort: external, InternalPort: info.internal, Protocol: protocol})
	}
	return rules, nil
}

func normalizeSwitcherConfig(cfg SwitcherConfig) SwitcherConfig {
	if cfg.PortRangeStart <= 0 {
		cfg.PortRangeStart = 30000
	}
	if cfg.PortRangeEnd <= 0 {
		cfg.PortRangeEnd = 40000
	}
	if cfg.PortRangeEnd < cfg.PortRangeStart {
		cfg.PortRangeEnd = cfg.PortRangeStart
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 10
	}
	if cfg.HealthTimeout <= 0 {
		cfg.HealthTimeout = defaultHealthTimeout
	}
	if cfg.HealthInterval <= 0 {
		cfg.HealthInterval = defaultHealthInterval
	}
	if cfg.DrainTimeout <= 0 {
		cfg.DrainTimeout = defaultDrainTimeout
	}
	if strings.TrimSpace(cfg.NftBin) == "" {
		cfg.NftBin = "/usr/sbin/nft"
	}
	if strings.TrimSpace(cfg.ConntrackBin) == "" {
		cfg.ConntrackBin = "conntrack"
	}
	if strings.TrimSpace(cfg.NftTableName) == "" {
		cfg.NftTableName = "xboard_proxy"
	}
	if strings.TrimSpace(cfg.PIDDir) == "" {
		cfg.PIDDir = defaultPIDDir
	}
	if strings.TrimSpace(cfg.CgroupBasePath) == "" {
		cfg.CgroupBasePath = defaultCgroupBase
	}
	return cfg
}

func internalPortsFromMappings(externalPorts []int, mappings map[int]int) []int {
	ports := make([]int, 0, len(externalPorts))
	for _, external := range externalPorts {
		internal := mappings[external]
		if internal > 0 {
			ports = append(ports, internal)
		}
	}
	return ports
}

func hasMappingConflict(occupied map[int]bool, ports []int) bool {
	if len(ports) == 0 {
		return true
	}
	seen := make(map[int]bool, len(ports))
	for _, port := range ports {
		if port <= 0 {
			return true
		}
		if seen[port] {
			return true
		}
		seen[port] = true
		if occupied != nil && occupied[port] {
			return true
		}
	}
	return false
}

func clonePorts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := make([]int, len(values))
	copy(out, values)
	return out
}

func cloneGroup(group *InstanceGroup) *InstanceGroup {
	if group == nil {
		return nil
	}
	return &InstanceGroup{
		ID:            group.ID,
		ExternalPorts: clonePorts(group.ExternalPorts),
		InternalPorts: clonePorts(group.InternalPorts),
		CoreType:      group.CoreType,
		InstanceID:    group.InstanceID,
	}
}

func sanitizeToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, value)
}
