import { apiClient } from './client';

export const authApi = {
  login: async (email: string, password: string) => {
    const { data } = await apiClient.post<{ data: { access_token: string } }>('/auth/login', { email, password });
    return data.data;
  },
  
  register: async (email: string, password: string) => {
    const { data } = await apiClient.post<{ data: { id: string; email: string } }>('/auth/register', { email, password });
    return data.data;
  }
};
