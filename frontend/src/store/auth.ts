import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { User } from '@/lib/api/auth';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      setAuth: (user, accessToken, refreshToken) => {
        localStorage.setItem('access_token', accessToken);
        localStorage.setItem('refresh_token', refreshToken);
        set({ user, accessToken, refreshToken });
      },
      clearAuth: () => {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        set({ user: null, accessToken: null, refreshToken: null });
      },
      isAuthenticated: () => {
        const state = get();
        // Check both state and localStorage for consistency
        const tokenFromStorage = localStorage.getItem('access_token');
        const hasToken = !!(state.accessToken || tokenFromStorage);
        const hasUser = !!state.user;
        
        // If we have a token in localStorage but not in state, sync it
        if (tokenFromStorage && !state.accessToken) {
          const refreshToken = localStorage.getItem('refresh_token');
          set({ 
            accessToken: tokenFromStorage, 
            refreshToken: refreshToken 
          });
        }
        
        // For initial authentication check, we only need a valid token
        // The user will be loaded after authentication is confirmed
        return hasToken;
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
      }),
    }
  )
);

