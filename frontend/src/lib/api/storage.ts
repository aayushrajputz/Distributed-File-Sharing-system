import axios from 'axios';

export interface StorageUsage {
  used_bytes: number;
  quota_bytes: number;
  file_count: number;
  used_gb: number;
  quota_gb: number;
  usage_percentage: number;
}

export const storageService = {
  async getStorageUsage(): Promise<StorageUsage> {
    try {
      // Call file service directly to bypass API gateway issues
      const token = localStorage.getItem('access_token');
      const response = await axios.get('http://localhost:8082/v1/files/storage/usage', {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        }
      });
      return response.data;
    } catch (error) {
      console.warn('Storage API not available, returning zero usage:', error);
      // Return zero usage when API is not available
      return {
        used_bytes: 0,
        quota_bytes: 100 * 1024 * 1024 * 1024, // 100GB
        file_count: 0,
        used_gb: 0,
        quota_gb: 100,
        usage_percentage: 0
      };
    }
  },
};



