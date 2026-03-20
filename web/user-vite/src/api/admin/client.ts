import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from "axios";
import { getToken, clearToken, redirectToLogin } from "@/lib/auth";
import { ADMIN_API_VERSION, ADMIN_AUTH_ROUTES, ROUTES } from "@/lib/constants";

// Get base URL for admin API
const getAdminBaseURL = (): string => {
  const settings = window?.settings;
  const baseURL = settings?.base_url || import.meta.env.VITE_API_BASE_URL || "";
  return baseURL.replace(/\/$/, "") + ADMIN_API_VERSION;
};

type AdminApiErrorPayload = {
  error?: string;
  message?: string;
  action?: string;
  details?: unknown;
};

export class AdminApiError extends Error {
  status?: number;
  action?: string;
  details?: unknown;

  constructor(
    message: string,
    options: {
      status?: number;
      action?: string;
      details?: unknown;
    } = {}
  ) {
    super(message);
    this.name = "AdminApiError";
    this.status = options.status;
    this.action = options.action;
    this.details = options.details;
    Object.setPrototypeOf(this, AdminApiError.prototype);
  }
}

export function isAdminApiError(error: unknown): error is AdminApiError {
  return error instanceof AdminApiError;
}

// Admin API instance (requires admin authentication)
export const adminApi: AxiosInstance = axios.create({
  baseURL: getAdminBaseURL(),
  timeout: 15000,
  withCredentials: true,
});

// Request interceptor: inject token
adminApi.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getToken();
  if (token) {
    config.headers = config.headers || {};
    config.headers.Authorization = token.startsWith("Bearer") ? token : `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: handle errors
adminApi.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status;

    // Unauthorized or Forbidden, redirect to login
    if (status === 401) {
      clearToken();
      redirectToLogin(ADMIN_AUTH_ROUTES.LOGIN);
      return Promise.reject(error);
    }

    // Forbidden (not admin)
    if (status === 403) {
      window.location.href = ROUTES.DASHBOARD;
      return Promise.reject(error);
    }

    const payload = error?.response?.data as AdminApiErrorPayload | undefined;
    const message = payload?.error || payload?.message || error.message || "Request failed";
    return Promise.reject(
      new AdminApiError(message, {
        status,
        action: payload?.action,
        details: payload?.details,
      })
    );
  }
);
