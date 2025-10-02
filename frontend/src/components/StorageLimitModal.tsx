'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { billingService, Usage } from '@/lib/api/billing'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { 
  AlertTriangle, 
  HardDrive, 
  TrendingUp, 
  X, 
  Check,
  Zap,
  Building
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface StorageLimitModalProps {
  isOpen: boolean
  onClose: () => void
  userId: string
  fileSize?: number
}

export function StorageLimitModal({ 
  isOpen, 
  onClose, 
  userId, 
  fileSize = 0 
}: StorageLimitModalProps) {
  const router = useRouter()
  const [usage, setUsage] = useState<Usage | null>(null)
  const [plans, setPlans] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  const loadData = async () => {
    if (!isOpen) return
    
    try {
      setLoading(true)
      const [usageResponse, plansResponse] = await Promise.all([
        billingService.getUsage(userId),
        billingService.getPlans()
      ])
      setUsage(usageResponse.usage)
      setPlans(plansResponse.plans.filter(plan => plan.price_per_month > 0))
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleUpgrade = async (planId: string) => {
    try {
      const response = await billingService.createSubscription({
        user_id: userId,
        plan_id: planId,
        payment_method: 'stripe'
      })
      
      // Redirect to Stripe checkout
      window.location.href = response.payment_url
    } catch (error) {
      console.error('Failed to create subscription:', error)
    }
  }

  const getPlanIcon = (planName: string) => {
    switch (planName.toLowerCase()) {
      case 'pro':
        return <Zap className="h-5 w-5" />
      case 'enterprise':
        return <Building className="h-5 w-5" />
      default:
        return <TrendingUp className="h-5 w-5" />
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-4">
            <div className="p-3 bg-red-100 rounded-full">
              <AlertTriangle className="h-8 w-8 text-red-600" />
            </div>
          </div>
          <CardTitle className="text-2xl text-red-600">Storage Limit Reached</CardTitle>
          <CardDescription className="text-lg">
            You&apos;ve reached your storage limit and cannot upload more files.
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Current Usage */}
          {usage && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <HardDrive className="h-5 w-5 text-blue-600" />
                <span className="font-medium">Current Usage</span>
              </div>
              
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>{usage.used_gb.toFixed(1)} GB used</span>
                  <span>{usage.quota_gb} GB total</span>
                </div>
                <Progress value={usage.percent_used} className="h-3" />
                <div className="text-center text-sm text-muted-foreground">
                  {usage.percent_used.toFixed(1)}% of storage used
                </div>
              </div>

              {fileSize > 0 && (
                <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <p className="text-sm text-yellow-800">
                    <strong>File size:</strong> {formatBytes(fileSize)}<br />
                    <strong>Available space:</strong> {formatBytes(usage.quota_bytes - usage.used_bytes)}
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Upgrade Options */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-center">Upgrade Your Plan</h3>
            
            {loading ? (
              <div className="flex justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {plans.map((plan) => (
                  <Card 
                    key={plan.id}
                    className={cn(
                      "cursor-pointer transition-all duration-200 hover:shadow-md",
                      plan.is_popular ? "border-2 border-blue-500" : "border border-gray-200"
                    )}
                    onClick={() => handleUpgrade(plan.id)}
                  >
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {getPlanIcon(plan.name)}
                          <span className="font-semibold">{plan.name}</span>
                        </div>
                        {plan.is_popular && (
                          <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">
                            Popular
                          </span>
                        )}
                      </div>
                      <div className="text-2xl font-bold">
                        ${plan.price_per_month}
                        <span className="text-sm font-normal text-muted-foreground">/month</span>
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <div className="space-y-2">
                        <div className="flex items-center gap-2 text-sm">
                          <HardDrive className="h-4 w-4 text-blue-600" />
                          <span>{(plan.quota_bytes / (1024 * 1024 * 1024)).toFixed(0)} GB Storage</span>
                        </div>
                        {plan.features.slice(0, 3).map((feature: string, index: number) => (
                          <div key={index} className="flex items-center gap-2 text-sm">
                            <Check className="h-4 w-4 text-green-600" />
                            <span>{feature}</span>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex gap-3 pt-4">
            <Button
              variant="outline"
              onClick={onClose}
              className="flex-1"
            >
              <X className="h-4 w-4 mr-2" />
              Cancel
            </Button>
            <Button
              onClick={() => router.push('/billing')}
              className="flex-1"
            >
              <TrendingUp className="h-4 w-4 mr-2" />
              View All Plans
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}