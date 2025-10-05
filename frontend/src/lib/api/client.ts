import axios, { AxiosInstance } from 'axios';

export const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
// Use API Gateway for all requests
const authServiceUrl = `${apiGatewayUrl}/api/v1/auth`;
const fileServiceUrl = `${apiGatewayUrl}/api/v1/files`;
const notificationServiceUrl = `${apiGatewayUrl}/api/v1/notifications`;
const billingServiceUrl = `${apiGatewayUrl}/api/v1/billing`;

// Create axios instances for each service
export const authApi: AxiosInstance = axios.create({
  baseURL: authServiceUrl,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const fileApi: AxiosInstance = axios.create({
  baseURL: fileServiceUrl,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const notificationApi: AxiosInstance = axios.create({
  baseURL: notificationServiceUrl,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const billingApi: AxiosInstance = axios.create({
  baseURL: billingServiceUrl,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
const addAuthInterceptor = (api: AxiosInstance) => {
  api.interceptors.request.use(
    (config) => {
      const token = localStorage.getItem('access_token');
      console.log('Auth interceptor - token found:', !!token, 'URL:', config.url);
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
        console.log('Auth interceptor - Authorization header set');
      } else {
        console.log('Auth interceptor - No token found in localStorage');
      }
      return config;
    },
    (error) => Promise.reject(error)
  );
};

// Response interceptor to handle auth errors
const addResponseInterceptor = (api: AxiosInstance) => {
  api.interceptors.response.use(
    (response) => response,
    (error) => {
      // Log detailed error information
      if (error.response) {
        console.error('API Error Response:', {
          status: error.response.status,
          statusText: error.response.statusText,
          data: error.response.data,
          url: error.config?.url,
          method: error.config?.method,
        });
      } else if (error.request) {
        console.error('API No Response:', {
          url: error.config?.url,
          method: error.config?.method,
          message: 'No response received from server',
        });
      } else {
        console.error('API Request Error:', error.message);
      }

      if (error.response?.status === 401) {
        // Only clear auth and redirect if we're not already on the login page
        if (typeof window !== 'undefined' && window.location.pathname !== '/auth/login') {
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          window.location.href = '/auth/login';
        }
      }
      return Promise.reject(error);
    }
  );
};

// Apply interceptors
[authApi, fileApi, notificationApi, billingApi].forEach((api) => {
  addAuthInterceptor(api);
  addResponseInterceptor(api);
});

