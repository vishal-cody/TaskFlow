import { apiClient } from './client';

export interface Job {
  id: string;
  type: string;
  status: 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled' | 'retrying';
  priority: number;
  progress: number;
  payload: Record<string, unknown>;
  result: Record<string, unknown> | null;
  error: string;
  retry_count: number;
  max_retries: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface JobLog {
  id: number;
  job_id: string;
  level: string;
  message: string;
  timestamp: string;
}

export interface JobsListResponse {
  jobs: Job[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface StatsResponse {
  [status: string]: number;
}

export const jobsApi = {
  list: async (page = 1, limit = 10, status?: string) => {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });
    if (status) params.append('status', status);
    
    const { data } = await apiClient.get<{ data: JobsListResponse }>('/jobs', { params });
    return data.data;
  },
  
  get: async (id: string) => {
    const { data } = await apiClient.get<{ data: Job }>(`/jobs/${id}`);
    return data.data;
  },
  
  create: async (payload: { type: string; priority: number; payload: Record<string, unknown> }) => {
    const { data } = await apiClient.post<{ data: { id: string; status: string } }>('/jobs', payload);
    return data.data;
  },
  
  cancel: async (id: string) => {
    const { data } = await apiClient.post(`/jobs/${id}/cancel`);
    return data;
  },
  
  logs: async (id: string) => {
    const { data } = await apiClient.get<{ data: { logs: JobLog[] } }>(`/jobs/${id}/logs`);
    return data.data.logs;
  },
  
  stats: async () => {
    const { data } = await apiClient.get<{ data: StatsResponse }>('/jobs/stats');
    return data.data;
  }
};
