import { GetLoginStatus } from '@/services/Auth';

export async function requireAuth(
  _location?: { pathname: string },
  _customMessage?: string,
  _isError?: boolean
) {
  // Auth guard and redirect are handled by the /__requires_login layout.
  // This function is kept for compatibility but is effectively a no-op
  // since the layout's beforeLoad runs first.
  const response = await GetLoginStatus();
  return response;
}
