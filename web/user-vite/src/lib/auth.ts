const TOKEN_KEY = "xboard-token";
const REFRESH_TOKEN_KEY = "xboard-refresh-token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setRefreshToken(token: string): void {
  localStorage.setItem(REFRESH_TOKEN_KEY, token);
}

const normalizePathname = (path: string): string => {
  if (!path) {
    return "/";
  }
  const trimmed = path.replace(/\/+$/, "");
  return trimmed || "/";
};

export function isSamePath(pathname: string, targetPath: string): boolean {
  return normalizePathname(pathname) === normalizePathname(targetPath);
}

export function redirectToLogin(loginPath: string): void {
  const returnUrl = encodeURIComponent(window.location.pathname + window.location.search);
  if (!isSamePath(window.location.pathname, loginPath)) {
    window.location.href = `${loginPath}?next=${returnUrl}`;
  }
}
