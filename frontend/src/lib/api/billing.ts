import { apiGatewayUrl } from './client';

const billingApiUrl = `${apiGatewayUrl}/api/v1/billing`;

// Types
export interface Plan {
  id: string;
  name: string;
  quota_bytes: number;
  price_per_month: number;
  description: string;
  features: string[];
  is_popular: boolean;
  created_at: string;
  updated_at: string;
}

export interface Subscription {
  id: string;
  user_id: string;
  plan_id: string;
  plan?: Plan;
  status: 'active' | 'expired' | 'cancelled' | 'pending';
  payment_status: 'pending' | 'paid' | 'failed' | 'refunded';
  start_date: string;
  end_date: string;
  transaction_id?: string;
  payment_method: string;
  created_at: string;
  updated_at: string;
}

export interface Usage {
  user_id: string;
  plan_name: string;
  quota_bytes: number;
  used_bytes: number;
  quota_gb: number;
  used_gb: number;
  percent_used: number;
  upgrade_available: boolean;
  quota_exceeded: boolean;
}

export interface CreateSubscriptionRequest {
  user_id: string;
  plan_id: string;
  payment_method: 'stripe' | 'razorpay';
}

export interface CreateSubscriptionResponse {
  subscription: Subscription;
  payment_url: string;
  client_secret: string;
  session_id: string;
}

export interface CancelSubscriptionRequest {
  user_id: string;
  subscription_id: string;
}

export interface CancelSubscriptionResponse {
  success: boolean;
  message: string;
}

export interface GetUserSubscriptionResponse {
  subscription?: Subscription;
  has_active_subscription: boolean;
}

// API Client
export const billingService = {
  // Get all available plans
  async getPlans(): Promise<{ plans: Plan[] }> {
    const response = await fetch(`${billingApiUrl}/plans`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch plans: ${response.statusText}`);
    }

    return response.json();
  },

  // Get a specific plan
  async getPlan(planId: string): Promise<{ plan: Plan }> {
    const response = await fetch(`${billingApiUrl}/plans/${planId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch plan: ${response.statusText}`);
    }

    return response.json();
  },

  // Get user's current subscription
  async getUserSubscription(userId: string): Promise<GetUserSubscriptionResponse> {
    const response = await fetch(`${billingApiUrl}/subscription?user_id=${userId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch subscription: ${response.statusText}`);
    }

    return response.json();
  },

  // Create a new subscription
  async createSubscription(data: CreateSubscriptionRequest): Promise<CreateSubscriptionResponse> {
    const response = await fetch(`${billingApiUrl}/subscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error(`Failed to create subscription: ${response.statusText}`);
    }

    return response.json();
  },

  // Cancel a subscription
  async cancelSubscription(data: CancelSubscriptionRequest): Promise<CancelSubscriptionResponse> {
    const response = await fetch(`${billingApiUrl}/subscription/cancel`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error(`Failed to cancel subscription: ${response.statusText}`);
    }

    return response.json();
  },

  // Get storage usage
  async getUsage(userId: string): Promise<{ usage: Usage }> {
    try {
      // Try to get usage from file service first
      const response = await fetch(`${process.env.NEXT_PUBLIC_FILE_SERVICE_URL}/v1/files/storage/usage`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        // Convert file service response to billing service format
        return {
          usage: {
            user_id: userId,
            plan_name: 'Pro', // Default plan
            quota_bytes: data.quota_bytes || (100 * 1024 * 1024 * 1024), // 100GB default
            used_bytes: data.used_bytes || 0,
            quota_gb: data.quota_gb || 100,
            used_gb: data.used_gb || 0,
            percent_used: data.usage_percentage || 0,
            upgrade_available: true,
            quota_exceeded: (data.used_bytes || 0) >= (data.quota_bytes || 100 * 1024 * 1024 * 1024)
          }
        };
      }
    } catch (error) {
      console.warn('File service not available, trying billing service:', error);
    }

    // Fallback to billing service
    const response = await fetch(`${billingApiUrl}/usage?user_id=${userId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch usage: ${response.statusText}`);
    }

    return response.json();
  },
};