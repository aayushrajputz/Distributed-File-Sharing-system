import { fileApi } from './client';
import axios from 'axios';

// Helper function to get user ID from auth store
const getUserId = (): string => {
  // Try to get from Zustand persist storage first
  const authStorage = localStorage.getItem('auth-storage');
  if (authStorage) {
    try {
      const parsed = JSON.parse(authStorage);
      const userId = parsed.state?.user?.userId || parsed.state?.user?.user_id;
      if (userId) return userId;
    } catch (e) {
      console.error('Failed to parse auth-storage:', e);
    }
  }

  // Fallback to old user storage
  const userStr = localStorage.getItem('user');
  if (userStr) {
    try {
      const user = JSON.parse(userStr);
      return user.userId || user.user_id || '';
    } catch (e) {
      console.error('Failed to parse user from localStorage:', e);
    }
  }

  return '';
};

export interface FileMetadata {
  id: string;
  file_id: string;
  name: string;
  description?: string;
  size: number;
  mime_type: string;
  owner_id: string;
  storage_path: string;
  checksum?: string;
  status: string;
  created_at: string;
  updated_at: string;
  is_private?: boolean;
  shared_with?: string[];
}

export interface UploadFileRequest {
  name: string;
  description?: string;
  size: number;
  mime_type: string;
  is_private?: boolean;
}

export interface FileShare {
  share_id: string;
  file_id: string;
  owner_id: string;
  shared_with_email: string;
  permission: string;
  expiry_time?: string;
  share_link?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export const fileService = {
  async uploadFile(data: UploadFileRequest): Promise<{ file_id: string; upload_url: string; message: string }> {
    console.log('Calling uploadFile API with data:', data);
    const response = await fileApi.post('/upload', data);
    console.log('Upload API response:', {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers,
      data: response.data,
    });

    // Handle both camelCase and snake_case response formats
    const responseData = response.data;
    return {
      file_id: responseData.file_id || responseData.fileId || '',
      upload_url: responseData.upload_url || responseData.uploadUrl || '',
      message: responseData.message || '',
    };
  },

  async uploadToStorage(uploadUrl: string, file: File, onProgress?: (progress: number) => void): Promise<void> {
    console.log('Uploading to MinIO presigned URL:', uploadUrl);
    console.log('File details:', { name: file.name, size: file.size, type: file.type });

    // Use a clean axios instance without interceptors to avoid modifying the presigned URL signature
    const cleanAxios = axios.create();

    try {
      await cleanAxios.put(uploadUrl, file, {
        headers: {
          'Content-Type': file.type || 'application/octet-stream',
        },
        onUploadProgress: (progressEvent) => {
          if (onProgress && progressEvent.total) {
            const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
            onProgress(progress);
          }
        },
      });
      console.log('Upload to MinIO successful');
    } catch (error) {
      console.error('Upload to MinIO failed:', error);
      throw error;
    }
  },

  async completeUpload(fileId: string, checksum?: string): Promise<{ file: FileMetadata; message: string }> {
    const response = await fileApi.post(`/${fileId}/complete`, {
      file_id: fileId,
      checksum,
    });
    return response.data;
  },

  async listFiles(page = 1, limit = 20): Promise<{ files: FileMetadata[]; total: number; page: number; limit: number }> {
    console.log('listFiles - page:', page, 'limit:', limit);
    console.log('listFiles - localStorage access_token:', !!localStorage.getItem('access_token'));

    const response = await fileApi.get('/', {
      params: {
        page,
        limit
      },
    });
    console.log('listFiles - response:', response.status, response.data);
    return response.data;
  },

  async getFile(fileId: string): Promise<{ file: FileMetadata }> {
    const response = await fileApi.get(`/${fileId}`);
    return response.data;
  },

  async getDownloadUrl(fileId: string): Promise<{ download_url: string; expires_in: number }> {
    const response = await fileApi.get(`/${fileId}/download`);
    return response.data;
  },

  async deleteFile(fileId: string): Promise<{ message: string }> {
    // Delete file permanently - no trash functionality
    const response = await fileApi.delete(`/${fileId}/permanent`);
    return response.data;
  },

  async shareFile(fileId: string, emails: string[], permission: string, expiryTime?: string | null): Promise<{ shares: FileShare[]; share_link?: string; message: string }> {
    const response = await fileApi.post(`/${fileId}/share`, {
      file_id: fileId,
      shared_with_emails: emails,
      permission: permission.toUpperCase(),
      expiry_time: expiryTime,
    });
    return response.data;
  },

  async listSharedFiles(page = 1, limit = 20): Promise<{ files: FileMetadata[]; total: number; page: number; limit: number }> {
    console.log('listSharedFiles - page:', page, 'limit:', limit);

    const response = await fileApi.get('/shared', {
      params: {
        page,
        limit
      },
    });
    return response.data;
  },

  async addToFavorites(fileId: string): Promise<{ message: string; is_favorite: boolean }> {
    const response = await fileApi.post(`/${fileId}/favorite`);
    return response.data;
  },

  async removeFromFavorites(fileId: string): Promise<{ message: string; is_favorite: boolean }> {
    const response = await fileApi.delete(`/${fileId}/favorite`);
    return response.data;
  },

  async listFavorites(page = 1, limit = 20): Promise<{ files: FileMetadata[]; total: number; page: number; limit: number }> {
    const response = await fileApi.get('/favorites', {
      params: {
        page,
        limit
      },
    });
    return response.data;
  },

  async checkFavoriteStatus(fileIds: string[]): Promise<{ [fileId: string]: boolean }> {
    // For now, we'll get all favorites and check which ones match
    // This could be optimized with a dedicated endpoint later
    try {
      const response = await this.listFavorites(1, 100); // Get all favorites (capped at 100)
      const favoriteFileIds = new Set(response.files.map(f => f.file_id));

      const result: { [fileId: string]: boolean } = {};
      fileIds.forEach(fileId => {
        result[fileId] = favoriteFileIds.has(fileId);
      });

      return result;
    } catch (error) {
      console.error('Failed to check favorite status:', error);
      // Return all false if there's an error
      const result: { [fileId: string]: boolean } = {};
      fileIds.forEach(fileId => {
        result[fileId] = false;
      });
      return result;
    }
  },

  async downloadFile(
    fileId: string,
    fileName: string,
    onProgress?: (progress: { percentage: number; loaded: number; total: number; speed: number; estimatedTime: number }) => void
  ): Promise<void> {
    const token = localStorage.getItem('access_token');
    if (!token) {
      throw new Error('No authentication token found');
    }

    // Use API Gateway URL for download
    const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
    const downloadUrl = `${apiGatewayUrl}/api/v1/files/${fileId}/download`;

    const response = await fetch(downloadUrl, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Download failed' }));
      throw new Error(errorData.error || `Download failed with status ${response.status}`);
    }

    const contentLength = response.headers.get('Content-Length');
    const total = contentLength ? parseInt(contentLength, 10) : 0;

    if (!response.body) {
      throw new Error('Response body is null');
    }

    const reader = response.body.getReader();
    const chunks: Uint8Array[] = [];
    let loaded = 0;
    let startTime = Date.now();
    let lastUpdateTime = startTime;

    while (true) {
      const { done, value } = await reader.read();

      if (done) {
        break;
      }

      chunks.push(value);
      loaded += value.length;

      // Calculate progress
      if (onProgress && total > 0) {
        const now = Date.now();
        const elapsed = (now - startTime) / 1000; // seconds
        const speed = loaded / elapsed; // bytes per second
        const remaining = total - loaded;
        const estimatedTime = remaining / speed; // seconds

        // Update progress every 100ms to avoid too many updates
        if (now - lastUpdateTime > 100) {
          onProgress({
            percentage: Math.round((loaded / total) * 100),
            loaded,
            total,
            speed,
            estimatedTime,
          });
          lastUpdateTime = now;
        }
      }
    }

    // Final progress update
    if (onProgress && total > 0) {
      onProgress({
        percentage: 100,
        loaded: total,
        total,
        speed: 0,
        estimatedTime: 0,
      });
    }

    // Create blob from chunks
    const blob = new Blob(chunks as BlobPart[]);

    // Create download link and trigger download
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
  },

  async updatePrivacy(fileId: string, isPrivate: boolean, sharedWith?: string[]): Promise<{ message: string; file_id: string; is_private: boolean; shared_with: string[] }> {
    const response = await fileApi.patch(`/${fileId}/privacy`, {
      is_private: isPrivate,
      shared_with: sharedWith || [],
    });
    return response.data;
  },

  async managePrivateAccess(fileId: string, userIds: string[], action: 'add' | 'remove'): Promise<{ message: string; file_id: string; action: string; users: string[] }> {
    const response = await fileApi.post(`/${fileId}/share-private`, {
      user_ids: userIds,
      action,
    });
    return response.data;
  },

  async listPrivateFiles(page = 1, limit = 20): Promise<{ files: FileMetadata[]; total: number; page: number; limit: number }> {
    const response = await fileApi.get('/private', {
      params: {
        page,
        limit,
      },
    });
    return response.data;
  },

};

