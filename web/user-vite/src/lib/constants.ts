export const API_VERSION = "/api/v1";
export const ADMIN_API_VERSION = "/api/v2/admin";

export const QUERY_KEYS = {
  USER: ["user"],
  USER_INFO: ["user", "info"],
  SERVERS: ["user", "servers"],
  PLANS: ["user", "plans"],
  TRAFFIC: ["user", "traffic"],
  KNOWLEDGE: ["user", "knowledge"],
  SHORT_LINKS: ["user", "shortlinks"],
  USER_NOTICE: ["user", "notice"],
  // Admin query keys
  ADMIN_AGENTS: ["admin", "agents"],
  ADMIN_USERS: ["admin", "users"],
  ADMIN_PLANS: ["admin", "plans"],
  ADMIN_NOTICES: ["admin", "notices"],
  ADMIN_KNOWLEDGE: ["admin", "knowledge"],
  ADMIN_SYSTEM: ["admin", "system"],
  ADMIN_FORWARDING: ["admin", "forwarding"],
  ADMIN_FORWARDING_LOGS: ["admin", "forwarding", "logs"],
  ADMIN_ACCESS_LOGS: ["admin", "access-logs"],
  ADMIN_ACCESS_LOG_STATS: ["admin", "access-logs", "stats"],
  ADMIN_AGENT_CORES: ["admin", "agents", "cores"],
  ADMIN_AGENT_CORE_INSTANCES: ["admin", "agents", "core-instances"],
  ADMIN_AGENT_CORE_SWITCH_LOGS: ["admin", "agents", "core-switch-logs"],
} as const;

export const ROUTES = {
  INSTALL: "/install",
  LOGIN: "/login",
  REGISTER: "/register",
  FORGOT_PASSWORD: "/forgot-password",
  DASHBOARD: "/dashboard",
  SERVERS: "/servers",
  PLANS: "/plans",
  TRAFFIC: "/traffic",
  KNOWLEDGE: "/knowledge",
  SETTINGS: "/settings",
} as const;

export const ADMIN_ROUTES = {
  AGENTS: "/admin/agents",
  USERS: "/admin/users",
  PLANS: "/admin/plans",
  NOTICES: "/admin/notices",
  KNOWLEDGE: "/admin/knowledge",
  SYSTEM: "/admin/system",
  FORWARDING: "/admin/forwarding",
  ACCESS_LOGS: "/admin/access-logs",
} as const;
