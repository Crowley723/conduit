import { api, ApiError } from '@/services/Api';
import { POST_LOGIN_REDIRECT_KEY } from '@/constants/SessionStorage.ts';
import {
  getSessionStorage,
  removeSessionStorage,
  setSessionStorage,
} from '@/services/SessionStorage.ts';

export interface User {
  sub: string;
  iss: string;
  username: string;
  display_name: string;
  email: string;
  groups: string[];
}

export interface MTLSConfig {
  enabled: boolean;
}

export interface Config {
  mtls?: MTLSConfig;
}

export interface AuthResponse {
  authenticated: boolean;
  user: User;
  config?: Config;
}

export const authRedirect = {
  save: (path: string): void => {
    setSessionStorage(POST_LOGIN_REDIRECT_KEY, path);
  },

  get: (): string | null => {
    return getSessionStorage(POST_LOGIN_REDIRECT_KEY);
  },

  clear: (): void => {
    removeSessionStorage(POST_LOGIN_REDIRECT_KEY);
  },
};

export function Login(redirectTo?: string): void {
  if (redirectTo) {
    authRedirect.save(redirectTo);
  }
  const API_BASE = import.meta.env.VITE_API_URL ?? '';
  window.location.href = `${API_BASE}/api/auth/login`;
}

export async function Logout(): Promise<void> {
  return api.post<void>('/auth/logout');
}

export async function GetLoginStatus(): Promise<AuthResponse | null> {
  try {
    return await api.get<AuthResponse>('/auth/status');
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      return null;
    }
    throw e;
  }
}
