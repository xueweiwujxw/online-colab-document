import { apiBaseUrl, getJSON, sendJSON } from './client';

export type CurrentUser = {
  id: string;
  email: string;
  displayName: string;
  authSource: string;
  isAdmin: boolean;
};

export type LoginInput = {
  email: string;
  password: string;
};

export type RegisterInput = LoginInput & {
  displayName: string;
};

export type ChangePasswordInput = {
  currentPassword: string;
  newPassword: string;
};

export type UpdateProfileInput = {
  displayName: string;
};

export type AuthSession = {
  id: string;
  current: boolean;
  expiresAt: string;
  createdAt: string;
};

export function login(input: LoginInput): Promise<CurrentUser> {
  return sendJSON<CurrentUser>('/api/auth/local/login', input);
}

export function register(input: RegisterInput): Promise<CurrentUser> {
  return sendJSON<CurrentUser>('/api/auth/local/register', input);
}

export function logout(): Promise<{ status: string }> {
  return sendJSON<{ status: string }>('/api/auth/logout');
}

export function getCurrentUser(): Promise<CurrentUser> {
  return getJSON<CurrentUser>('/api/auth/me');
}

export function changePassword(input: ChangePasswordInput): Promise<{ status: string }> {
  return sendJSON<{ status: string }>('/api/auth/password', input, {
    method: 'PUT',
  });
}

export function updateProfile(input: UpdateProfileInput): Promise<CurrentUser> {
  return sendJSON<CurrentUser>('/api/auth/profile', input, {
    method: 'PUT',
  });
}

export async function listSessions(): Promise<AuthSession[]> {
  const response = await getJSON<{ items: AuthSession[] }>('/api/auth/sessions');
  return response.items;
}

export function revokeSession(id: string): Promise<{ status: string }> {
  return sendJSON<{ status: string }>(`/api/auth/sessions/${id}`, undefined, {
    method: 'DELETE',
  });
}

export function getOIDCLoginURL(): string {
  return `${apiBaseUrl}/api/auth/oidc/login`;
}
