// 文件路径: internal/repository/sqlite/store.go
// 模块说明: 这是 internal 模块里的 store 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package sqlite

import (
	"database/sql"

	"github.com/creamcroissant/xboard/internal/repository"
)

// Store wires SQLite-backed repository implementations.
type Store struct {
	db                     *sql.DB
	coreOperations         repository.CoreOperationRepository
	users                  repository.UserRepository
	settings               repository.SettingRepository
	invites                repository.InviteCodeRepository
	plugins                repository.PluginRepository
	plans                  repository.PlanRepository
	loginLogs              repository.LoginLogRepository
	tokens                 repository.TokenRepository
	servers                repository.ServerRepository
	groups                 repository.ServerGroupRepository
	routes                 repository.ServerRouteRepository
	statUsers              repository.StatUserRepository
	statServers            repository.StatServerRepository
	notices                repository.NoticeRepository
	knowledge              repository.KnowledgeRepository
	subLogs                repository.SubscriptionLogRepository
	agentHosts             repository.AgentHostRepository
	configTemplates        repository.ConfigTemplateRepository
	serverClientConfigs    repository.ServerClientConfigRepository
	userTraffic            repository.UserTrafficRepository
	shortLinks             repository.ShortLinkRepository
	subscriptionTemplates  repository.SubscriptionTemplateRepository
	forwardingRules        repository.ForwardingRuleRepository
	forwardingRuleLogs     repository.ForwardingRuleLogRepository
	userNoticeReads        repository.UserNoticeReadsRepository
	agentCoreInstances     repository.AgentCoreInstanceRepository
	agentCoreSwitchLogs    repository.AgentCoreSwitchLogRepository
	accessLogs             repository.AccessLogRepository
	inboundSpecs           repository.InboundSpecRepository
	inboundSpecRevisions   repository.InboundSpecRevisionRepository
	desiredArtifacts       repository.DesiredArtifactRepository
	applyRuns              repository.ApplyRunRepository
	agentConfigInventories repository.AgentConfigInventoryRepository
	inboundIndexes         repository.InboundIndexRepository
	driftStates            repository.DriftStateRepository
}

// NewStore constructs a SQLite-backed repository store.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:                     db,
		coreOperations:         newCoreOperationRepo(db),
		users:                  &userRepo{db: db},
		settings:               &settingRepo{db: db},
		invites:                &inviteRepo{db: db},
		plugins:                &pluginRepo{db: db},
		plans:                  &planRepo{db: db},
		loginLogs:              &loginLogRepo{db: db},
		tokens:                 &tokenRepo{db: db},
		servers:                &serverRepo{db: db},
		groups:                 &serverGroupRepo{db: db},
		routes:                 &serverRouteRepo{db: db},
		statUsers:              &statUserRepo{db: db},
		statServers:            &statServerRepo{db: db},
		notices:                &noticeRepo{db: db},
		knowledge:              &knowledgeRepo{db: db},
		subLogs:                &subscriptionLogRepo{db: db},
		agentHosts:             newAgentHostRepo(db),
		configTemplates:        newConfigTemplateRepo(db),
		serverClientConfigs:    newServerClientConfigRepo(db),
		userTraffic:            newUserTrafficRepo(db),
		shortLinks:             NewShortLinkRepository(db),
		subscriptionTemplates:  newSubscriptionTemplateRepo(db),
		forwardingRules:        newForwardingRuleRepo(db),
		forwardingRuleLogs:     newForwardingRuleLogRepo(db),
		userNoticeReads:        newUserNoticeReadsRepo(db),
		agentCoreInstances:     newAgentCoreInstanceRepo(db),
		agentCoreSwitchLogs:    newAgentCoreSwitchLogRepo(db),
		accessLogs:             newAccessLogRepo(db),
		inboundSpecs:           newInboundSpecRepo(db),
		inboundSpecRevisions:   newInboundSpecRevisionRepo(db),
		desiredArtifacts:       newDesiredArtifactRepo(db),
		applyRuns:              newApplyRunRepo(db),
		agentConfigInventories: newAgentConfigInventoryRepo(db),
		inboundIndexes:         newInboundIndexRepo(db),
		driftStates:            newDriftStateRepo(db),
	}
}

func (s *Store) CoreOperations() repository.CoreOperationRepository {
	return s.coreOperations
}

func (s *Store) Users() repository.UserRepository {
	return s.users
}

func (s *Store) Settings() repository.SettingRepository {
	return s.settings
}

func (s *Store) InviteCodes() repository.InviteCodeRepository {
	return s.invites
}

func (s *Store) Plugins() repository.PluginRepository {
	return s.plugins
}

func (s *Store) Plans() repository.PlanRepository {
	return s.plans
}

func (s *Store) LoginLogs() repository.LoginLogRepository {
	return s.loginLogs
}

func (s *Store) Tokens() repository.TokenRepository {
	return s.tokens
}

func (s *Store) Servers() repository.ServerRepository {
	return s.servers
}

func (s *Store) ServerGroups() repository.ServerGroupRepository {
	return s.groups
}

func (s *Store) ServerRoutes() repository.ServerRouteRepository {
	return s.routes
}

func (s *Store) StatUsers() repository.StatUserRepository {
	return s.statUsers
}

func (s *Store) StatServers() repository.StatServerRepository {
	return s.statServers
}

func (s *Store) Notices() repository.NoticeRepository {
	return s.notices
}

func (s *Store) Knowledge() repository.KnowledgeRepository {
	return s.knowledge
}

func (s *Store) SubscriptionLogs() repository.SubscriptionLogRepository {
	return s.subLogs
}

func (s *Store) AgentHosts() repository.AgentHostRepository {
	return s.agentHosts
}

func (s *Store) ConfigTemplates() repository.ConfigTemplateRepository {
	return s.configTemplates
}

func (s *Store) ServerClientConfigs() repository.ServerClientConfigRepository {
	return s.serverClientConfigs
}

func (s *Store) UserTraffic() repository.UserTrafficRepository {
	return s.userTraffic
}

func (s *Store) ShortLinks() repository.ShortLinkRepository {
	return s.shortLinks
}

func (s *Store) SubscriptionTemplates() repository.SubscriptionTemplateRepository {
	return s.subscriptionTemplates
}

func (s *Store) ForwardingRules() repository.ForwardingRuleRepository {
	return s.forwardingRules
}

func (s *Store) ForwardingRuleLogs() repository.ForwardingRuleLogRepository {
	return s.forwardingRuleLogs
}

func (s *Store) UserNoticeReads() repository.UserNoticeReadsRepository {
	return s.userNoticeReads
}

func (s *Store) AgentCoreInstances() repository.AgentCoreInstanceRepository {
	return s.agentCoreInstances
}

func (s *Store) AgentCoreSwitchLogs() repository.AgentCoreSwitchLogRepository {
	return s.agentCoreSwitchLogs
}

func (s *Store) AccessLogs() repository.AccessLogRepository {
	return s.accessLogs
}

func (s *Store) InboundSpecs() repository.InboundSpecRepository {
	return s.inboundSpecs
}

func (s *Store) InboundSpecRevisions() repository.InboundSpecRevisionRepository {
	return s.inboundSpecRevisions
}

func (s *Store) DesiredArtifacts() repository.DesiredArtifactRepository {
	return s.desiredArtifacts
}

func (s *Store) ApplyRuns() repository.ApplyRunRepository {
	return s.applyRuns
}

func (s *Store) AgentConfigInventories() repository.AgentConfigInventoryRepository {
	return s.agentConfigInventories
}

func (s *Store) InboundIndexes() repository.InboundIndexRepository {
	return s.inboundIndexes
}

func (s *Store) DriftStates() repository.DriftStateRepository {
	return s.driftStates
}
