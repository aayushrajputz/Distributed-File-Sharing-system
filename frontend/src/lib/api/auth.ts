import { authApi } from './client';

export interface RegisterRequest {
  email: string;
  password: string;
  fullName: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface User {
  userId: string;
  email: string;
  fullName: string;
  avatarUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: User;
}

export const authService = {
  async register(data: RegisterRequest): Promise<{ user: User }> {
    const response = await authApi.post('/register', data);
    return response.data;
  },

  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await authApi.post('/login', data);
    return response.data;
  },

  async validateToken(token: string): Promise<{ valid: boolean; user_id: string; email: string }> {
    const response = await authApi.post('/validate', { token });
    return response.data;
  },

  async getUser(userId: string): Promise<{ user: User }> {
    const response = await authApi.get(`/user/${userId}`);
    return response.data;
  },

  async refreshToken(refreshToken: string): Promise<{ access_token: string; expires_in: number }> {
    const response = await authApi.post('/refresh', { refresh_token: refreshToken });
    return response.data;
  },

  async changePassword(userId: string, currentPassword: string, newPassword: string): Promise<{ message: string }> {
    const response = await authApi.post('/change-password', {
      user_id: userId,
      current_password: currentPassword,
      new_password: newPassword,
    });
    return response.data;
  },
};

