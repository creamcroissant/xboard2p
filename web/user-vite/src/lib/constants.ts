export const API_VERSION = "/api/v1";

const normalizePath = (value: string | undefined, fallback: string): string => {
  const normalized = (value || "").trim();
  if (!normalized) {
    return fallback;
  }
  const withLeadingSlash = normalized.startsWith("/") ? normalized : `/${normalized}`;
  return withLeadingSlash.replace(/\/+$/, "") || fallback;
};

const runtimeSecurePath = normalizePath(window?.settings?.secure_path, "/admin");
const runtimeRouterBase = normalizePath(window?.settings?.router_base, "/");

export const ADMIN_SECURE_PATH = runtimeSecurePath;
export const ADMIN_API_VERSION = `/api/v2${runtimeSecurePath}`;
export const ADMIN_ROUTE_BASE = runtimeRouterBase;

const normalizeRoute = (route: string): string => {
  if (ADMIN_ROUTE_BASE === "/") {
    return route;
  }
  return `${ADMIN_ROUTE_BASE}${route}`;
};

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
  ADMIN_CONFIG_CENTER_SPECS: ["admin", "config-center", "specs"],
  ADMIN_CONFIG_CENTER_SPEC_HISTORY: ["admin", "config-center", "specs", "history"],
  ADMIN_CONFIG_CENTER_ARTIFACTS: ["admin", "config-center", "artifacts"],
  ADMIN_CONFIG_CENTER_DIFF_TEXT: ["admin", "config-center", "diff", "text"],
  ADMIN_CONFIG_CENTER_DIFF_SEMANTIC: ["admin", "config-center", "diff", "semantic"],
  ADMIN_CONFIG_CENTER_APPLY_RUNS: ["admin", "config-center", "apply-runs"],
  ADMIN_CONFIG_CENTER_SNAPSHOT: ["admin", "config-center", "snapshot"],
  ADMIN_CONFIG_CENTER_DRIFT: ["admin", "config-center", "drift"],
  ADMIN_CONFIG_CENTER_RECOVER: ["admin", "config-center", "recover"],
} as const;

export const ROUTES = {
  INSTALL: normalizeRoute("/install"),
  LOGIN: normalizeRoute("/login"),
  REGISTER: normalizeRoute("/register"),
  FORGOT_PASSWORD: normalizeRoute("/forgot-password"),
  DASHBOARD: normalizeRoute("/dashboard"),
  SERVERS: normalizeRoute("/servers"),
  PLANS: normalizeRoute("/plans"),
  TRAFFIC: normalizeRoute("/traffic"),
  KNOWLEDGE: normalizeRoute("/knowledge"),
  SETTINGS: normalizeRoute("/settings"),
} as const;

const adminRoute = (path: string): string => normalizeRoute(`${ADMIN_SECURE_PATH}${path}`);

export const ADMIN_ROUTES = {
  AGENTS: adminRoute("/agents"),
  USERS: adminRoute("/users"),
  PLANS: adminRoute("/plans"),
  NOTICES: adminRoute("/notices"),
  KNOWLEDGE: adminRoute("/knowledge"),
  SYSTEM: adminRoute("/system"),
  FORWARDING: adminRoute("/forwarding"),
  ACCESS_LOGS: adminRoute("/access-logs"),
  CONFIG_CENTER: adminRoute("/config-center"),
} as const;

export const ADMIN_AUTH_ROUTES = {
  LOGIN: adminRoute("/login"),
  REGISTER: adminRoute("/register"),
  FORGOT_PASSWORD: adminRoute("/forgot-password"),
} as const;
