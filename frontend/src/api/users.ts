import { getJSON } from './client';

export type UserSearchItem = {
  id: string;
  email: string;
  displayName: string;
  authSource: string;
  isAdmin: boolean;
};

export type UserSearchResult = {
  items: UserSearchItem[];
  hasMore: boolean;
};

export async function searchUsers(query: string, offset = 0): Promise<UserSearchResult> {
  const params = new URLSearchParams();
  if (query.trim() !== '') {
    params.set('q', query.trim());
  }
  params.set('limit', '50');
  params.set('offset', String(Math.max(offset, 0)));
  return getJSON<UserSearchResult>(`/api/users?${params.toString()}`);
}
