export const apiBaseUrl =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') ?? '';

export class ApiError extends Error {
  status: number;

  constructor(status: number, message?: string) {
    super(message ?? `请求失败，状态码 ${status}`);
    this.name = 'ApiError';
    this.status = status;
  }
}

export function isForbidden(error: unknown): boolean {
  return error instanceof ApiError && error.status === 403;
}

export function errorMessage(error: unknown, fallback: string): string {
  if (isForbidden(error)) {
    return '无权限访问';
  }
  return error instanceof Error ? error.message : fallback;
}

export async function getJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    credentials: 'include',
    ...init,
    headers: {
      Accept: 'application/json',
      ...init?.headers,
    },
  });

  if (!response.ok) {
    throw new ApiError(response.status);
  }

  return response.json() as Promise<T>;
}

export async function sendJSON<T>(
  path: string,
  body?: unknown,
  init?: RequestInit,
): Promise<T> {
  return getJSON<T>(path, {
    method: 'POST',
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}
