import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from "axios";
import { getToken, clearToken } from "@/lib/auth";
import { ADMIN_API_VERSION, ROUTES } from "@/lib/constants";

// Get base URL for admin API
const getAdminBaseURL = (): string => {
  const settings = window?.settings;
  const baseURL = settings?.base_url || import.meta.env.VITE_API_BASE_URL || "";
  return baseURL.replace(/\/$/, "") + ADMIN_API_VERSION;
};

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
      const returnUrl = encodeURIComponent(window.location.pathname + window.location.search);
      const loginPath = ROUTES.LOGIN;
      if (!window.location.pathname.startsWith(loginPath)) {
        window.location.href = `${loginPath}?next=${returnUrl}`;
      }
      return Promise.reject(error);
    }

    // Forbidden (not admin)
    if (status === 403) {
      window.location.href = ROUTES.DASHBOARD;
      return Promise.reject(error);
    }

    const message = error?.response?.data?.message || error.message || "Request failed";
    return Promise.reject(new Error(message));
  }
);
