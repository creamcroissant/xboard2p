/**
 * Admin module type definitions
 */

// Agent status enum
export const AgentStatus = {
  Offline: 0,
  Online: 1,
  Warning: 2,
} as const;

export type AgentStatus = (typeof AgentStatus)[keyof typeof AgentStatus];

// Agent host (server probe) interface
export interface AgentHost {
  id: number;
  name: string;
  host: string;
  port: number;
  token?: string;
  template_id?: number;
  status: AgentStatus;
  cpu_used: number;
  mem_total: number;
  mem_used: number;
  disk_total: number;
  disk_used: number;
  upload_total: number;
  download_total: number;
  last_heartbeat_at: number;
  created_at: number;
  updated_at: number;
}

// Admin user interface (extended from UserProfile)
export interface AdminUser {
  id: number;
  email?: string;
  username?: string;
  uuid: string;
  token: string;
  plan_id?: number;
  plan_name?: string;
  group_id?: number;
  transfer_enable: number;
  transfer_used: number;
  u: number;
  d: number;
  expired_at?: number;
  is_admin: boolean;
  is_staff: boolean;
  status: number;
  banned: boolean;
  commission_balance: number;
  telegram_id?: number;
  created_at: number;
  updated_at: number;
}

// Admin plan interface
export interface AdminPlan {
  id: number;
  name: string;
  content?: string;
  transfer_enable: number;
  speed_limit?: number;
  device_limit?: number;
  group_id?: number;
  show: boolean;
  sell: boolean;
  renew: boolean;
  sort: number;
  reset_traffic_method?: number;
  reset_traffic_value?: number;
  created_at: number;
  updated_at: number;
}

// Admin notice interface
export interface AdminNotice {
  id: number;
  title: string;
  content: string;
  img_url?: string;
  show: boolean;
  popup: boolean;
  sort: number;
  created_at: number;
  updated_at: number;
}

// Admin knowledge interface (list)
export interface AdminKnowledgeSummary {
  id: number;
  language: string;
  category: string;
  title: string;
  show: boolean;
  updated_at: number;
}

// Admin knowledge interface (detail)
export interface AdminKnowledgeDetail {
  id: number;
  language: string;
  category: string;
  title: string;
  body: string;
  sort: number;
  show: boolean;
  created_at: number;
  updated_at: number;
}

// Agent core info
export interface AgentCoreInfo {
  type: string;
  version: string;
  installed: boolean;
  capabilities: string[];
}

// Agent core instance interface
export interface AgentCoreInstance {
  id: number;
  agent_host_id: number;
  instance_id: string;
  core_type: string;
  status: string;
  listen_ports: number[];
  config_template_id?: number;
  config_hash: string;
  started_at?: number;
  last_heartbeat_at?: number;
  error_message: string;
  created_at: number;
  updated_at: number;
}

export interface CreateAgentCoreInstanceRequest {
  core_type: string;
  instance_id: string;
  config_template_id?: number;
  config_json?: Record<string, unknown>;
}

export interface SwitchAgentCoreRequest {
  from_instance_id?: string;
  to_core_type: string;
  config_template_id?: number;
  config_json?: Record<string, unknown>;
  switch_id?: string;
  listen_ports?: number[];
  zero_downtime?: boolean;
}

export interface SwitchAgentCoreResult {
  success: boolean;
  new_instance_id?: string;
  message?: string;
  error?: string;
  switch_log_id?: number;
  completed_at?: number;
  from_instance_id?: string;
  to_core_type?: string;
}

export interface ConvertCoreConfigRequest {
  source_core: string;
  target_core: string;
  config_json: Record<string, unknown>;
}

export interface ConvertCoreConfigResult {
  config_json: unknown;
  warnings?: string[];
}

export interface AgentCoreSwitchLog {
  id: number;
  agent_host_id: number;
  from_instance_id?: string;
  from_core_type?: string;
  to_instance_id: string;
  to_core_type: string;
  operator_id?: number;
  status: string;
  detail: string;
  created_at: number;
  completed_at?: number;
}

export interface AgentCoreSwitchLogsParams {
  agent_host_id: number;
  status?: string;
  start_at?: number;
  end_at?: number;
  limit?: number;
  offset?: number;
}

export interface AgentCoreSwitchLogsResponse {
  logs: AgentCoreSwitchLog[];
  total: number;
}

// Admin knowledge save request
export interface AdminKnowledgeSaveRequest {
  id?: number;
  language: string;
  category: string;
  title: string;
  body: string;
  sort?: number;
  show?: boolean;
}

// System status interface
export interface SystemStatus {
  version: string;
  go_version: string;
  uptime: number;
  user_count: number;
  server_count: number;
  agent_count: number;
  online_agent_count: number;
}

// Pagination params
export interface PaginationParams {
  page?: number;
  page_size?: number;
}

// Paginated response
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

// Create/Update agent host request
export interface CreateAgentHostRequest {
  name: string;
  host: string;
  port: number;
  token?: string;
  template_id?: number;
}

export interface UpdateAgentHostRequest extends Partial<CreateAgentHostRequest> {
  id: number;
}

// Create/Update user request
export interface CreateUserRequest {
  email?: string;
  username?: string;
  password: string;
  plan_id?: number;
  group_id?: number;
  is_admin?: boolean;
  is_staff?: boolean;
  expired_at?: number;
  transfer_enable?: number;
}

export interface UpdateUserRequest {
  id: number;
  email?: string;
  username?: string;
  password?: string;
  plan_id?: number;
  group_id?: number;
  is_admin?: boolean;
  is_staff?: boolean;
  banned?: boolean;
  expired_at?: number;
  transfer_enable?: number;
}

// Create/Update plan request
export interface CreatePlanRequest {
  name: string;
  content?: string;
  transfer_enable: number;
  speed_limit?: number;
  device_limit?: number;
  group_id?: number;
  show?: boolean;
  sell?: boolean;
  renew?: boolean;
  sort?: number;
}

export interface UpdatePlanRequest extends Partial<CreatePlanRequest> {
  id: number;
}

// Create/Update notice request
export interface CreateNoticeRequest {
  title: string;
  content: string;
  img_url?: string;
  show?: boolean;
  popup?: boolean;
  sort?: number;
}

export interface UpdateNoticeRequest extends Partial<CreateNoticeRequest> {
  id: number;
}

// Forwarding rule interface
export interface ForwardingRule {
  id: number;
  agent_host_id: number;
  name: string;
  protocol: "tcp" | "udp" | "both";
  listen_port: number;
  target_address: string;
  target_port: number;
  enabled: boolean;
  priority: number;
  remark: string;
  version: number;
  created_at: number;
  updated_at: number;
}

export interface ForwardingRulesResponse {
  rules: ForwardingRule[];
  version: number;
}

export interface CreateForwardingRuleRequest {
  agent_host_id: number;
  name: string;
  protocol: "tcp" | "udp" | "both";
  listen_port: number;
  target_address: string;
  target_port: number;
  enabled: boolean;
  priority: number;
  remark: string;
}

export type UpdateForwardingRuleRequest = Partial<CreateForwardingRuleRequest>;

export interface ForwardingRuleLog {
  id: number;
  rule_id?: number;
  agent_host_id: number;
  action: "create" | "update" | "delete" | "apply" | "fail";
  operator_id?: number;
  detail: string;
  created_at: number;
}

export interface ForwardingRuleLogsResponse {
  logs: ForwardingRuleLog[];
  total: number;
}

export interface ForwardingRuleLogsParams {
  agent_host_id: number;
  rule_id?: number;
  start_at?: number;
  end_at?: number;
  limit?: number;
  offset?: number;
}

export interface AccessLogEntry {
  id: number;
  user_id?: number;
  user_email: string;
  agent_host_id: number;
  source_ip: string;
  target_domain: string;
  target_ip: string;
  target_port: number;
  protocol: string;
  upload: number;
  download: number;
  connection_start?: number;
  connection_end?: number;
  created_at: number;
}

export interface AccessLogQueryParams {
  limit?: number;
  offset?: number;
  user_id?: number;
  agent_host_id?: number;
  target_domain?: string;
  source_ip?: string;
  protocol?: string;
  start_at?: number;
  end_at?: number;
}

export interface AccessLogListResponse {
  logs: AccessLogEntry[];
  total: number;
}

export interface AccessLogStats {
  total_count: number;
  total_upload: number;
  total_download: number;
}

export interface AccessLogStatsParams {
  user_id?: number;
  agent_host_id?: number;
  start_at?: number;
  end_at?: number;
}
