import { getJSON } from './client';

export type UserSearchItem = {
  id: string;
  email: string;
  displayName: string;
  authSource: string;
  isAdmin: boolean;
};

export async function searchUsers(query: string): Promise<UserSearchItem[]> {
  const params = new URLSearchParams();
  if (query.trim() !== '') {
    params.set('q', query.trim());
  }
  params.set('limit', '20');
  const response = await getJSON<{ items: UserSearchItem[] }>(`/api/users?${params.toString()}`);
  return response.items;
}
