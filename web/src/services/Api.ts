export class ApiError extends Error {
  status: number;
  statusText: string;
  data?: unknown;

  constructor(status: number, statusText: string, data?: unknown) {
    super(`API Error ${status}: ${statusText}`);
    this.name = 'ApiError';
    this.status = status;
    this.statusText = statusText;
    this.data = data;
  }
}

interface RequestConfig extends RequestInit {
  params?: Record<string, string>;
}

async function requestImpl(
  endpoint: string,
  config?: RequestConfig
): Promise<unknown> {
  const { params, ...fetchConfig } = config || {};

  let url = `/api${endpoint}`;
  if (params) {
    const searchParams = new URLSearchParams(params);
    url += `?${searchParams.toString()}`;
  }

  const response = await fetch(url, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...fetchConfig.headers,
    },
    ...fetchConfig,
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new ApiError(response.status, response.statusText, data);
  }

  if (response.status === 204) {
    return undefined;
  }

  const contentType = response.headers.get('content-type');
  if (
    contentType?.includes('application/octet-stream') ||
    contentType?.includes('application/zip')
  ) {
    return response.blob();
  }

  return response.json();
}

export const api = {
  get<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
    return requestImpl(endpoint, { params }) as Promise<T>;
  },

  post<T>(endpoint: string, data?: unknown): Promise<T> {
    return requestImpl(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    }) as Promise<T>;
  },

  put<T>(endpoint: string, data?: unknown): Promise<T> {
    return requestImpl(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    }) as Promise<T>;
  },

  patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return requestImpl(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    }) as Promise<T>;
  },

  delete<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
    return requestImpl(endpoint, { method: 'DELETE', params }) as Promise<T>;
  },
};
