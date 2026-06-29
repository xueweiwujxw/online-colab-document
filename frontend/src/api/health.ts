import { getJSON } from './client';

export type HealthResponse = {
  status: string;
};

export function getHealth(): Promise<HealthResponse> {
  return getJSON<HealthResponse>('/healthz');
}
