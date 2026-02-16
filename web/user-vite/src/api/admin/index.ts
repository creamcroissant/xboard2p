// Admin API module exports
export { adminApi } from "./client";

// Agent Host API
export {
  getAgentHosts,
  getAgentHost,
  createAgentHost,
  updateAgentHost,
  deleteAgentHost,
  refreshAgentHosts,
} from "./agentHost";

// User API
export {
  getUsers,
  getUser,
  createUser,
  updateUser,
  deleteUser,
  toggleUserBan,
  resetUserPassword,
} from "./user";

// Plan API
export {
  getPlans,
  getPlan,
  createPlan,
  updatePlan,
  deletePlan,
  updatePlanSort,
} from "./plan";

// Notice API
export {
  getNotices,
  getNotice,
  createNotice,
  updateNotice,
  deleteNotice,
  toggleNoticeVisibility,
} from "./notice";

// Knowledge API
export {
  getKnowledgeList,
  getKnowledgeDetail,
  getKnowledgeCategories,
  saveKnowledgeArticle,
  toggleKnowledgeVisibility,
  deleteKnowledgeArticle,
  sortKnowledgeArticles,
} from "./knowledge";

// Forwarding API
export {
  listForwardingRules,
  createForwardingRule,
  updateForwardingRule,
  deleteForwardingRule,
  listForwardingLogs,
} from "./forwarding";

// Agent Core API
export {
  listAgentCores,
  listAgentCoreInstances,
  createAgentCoreInstance,
  deleteAgentCoreInstance,
  switchAgentCore,
  convertAgentCoreConfig,
  listAgentCoreSwitchLogs,
} from "./cores";

// Access Logs API
export { fetchAccessLogs, getAccessLogStats, cleanupAccessLogs } from "./accessLog";

// System API
export {
  getSystemStatus,
  getSystemConfig,
  updateSystemConfig,
} from "./system";
