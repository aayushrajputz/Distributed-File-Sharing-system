'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { billingService, Usage } from '@/lib/api/billing'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import { HardDrive, TrendingUp, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'

interface StorageUsageIndicatorProps {
  userId: string
  className?: string
  showUpgradeButton?: boolean
}

export function StorageUsageIndicator({ 
  userId, 
  className,
  showUpgradeButton = true 
}: StorageUsageIndicatorProps) {
  const router = useRouter()
  const [usage, setUsage] = useState<Usage | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadUsage()
  }, [userId])

  const loadUsage = async () => {
    try {
      const response = await billingService.getUsage(userId)
      setUsage(response.usage)
    } catch (error) {
      console.error('Failed to load storage usage:', error)
      // Set default values if API fails
      setUsage({
        user_id: userId,
        plan_name: 'Free',
        quota_bytes: 5 * 1024 * 1024 * 1024, // 5GB
        used_bytes: 0,
        quota_gb: 5,
        used_gb: 0,
        percent_used: 0,
        upgrade_available: true,
        quota_exceeded: false
      })
    } finally {
      setLoading(false)
    }
  }

  const getUsageColor = (percent: number) => {
    if (percent >= 95) return 'text-red-600'
    if (percent >= 80) return 'text-yellow-600'
    return 'text-green-600'
  }

  const getProgressColor = (percent: number) => {
    if (percent >= 95) return 'bg-red-500'
    if (percent >= 80) return 'bg-yellow-500'
    return 'bg-green-500'
  }

  if (loading) {
    return (
      <div className={cn("flex items-center space-x-2", className)}>
        <div className="animate-pulse bg-gray-200 rounded h-4 w-20"></div>
        <div className="animate-pulse bg-gray-200 rounded h-2 w-16"></div>
      </div>
    )
  }

  if (!usage) return null

  return (
    <div className={cn("flex items-center space-x-3", className)}>
      <div className="flex items-center space-x-2">
        <HardDrive className="h-4 w-4 text-blue-600" />
        <span className="text-sm font-medium">
          {usage.used_gb.toFixed(1)} GB
        </span>
        <span className="text-xs text-muted-foreground">
          of {usage.quota_gb} GB
        </span>
      </div>
      
      <div className="flex items-center space-x-2 min-w-[100px]">
        <Progress 
          value={usage.percent_used} 
          className="h-2 flex-1"
        />
        <span className={cn(
          "text-xs font-medium min-w-[3rem]",
          getUsageColor(usage.percent_used)
        )}>
          {usage.percent_used.toFixed(0)}%
        </span>
      </div>

      {usage.quota_exceeded && (
        <AlertTriangle className="h-4 w-4 text-red-500" />
      )}

      {showUpgradeButton && usage.upgrade_available && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => router.push('/billing')}
          className="text-xs h-7 px-2"
        >
          <TrendingUp className="h-3 w-3 mr-1" />
          Upgrade
        </Button>
      )}
    </div>
  )
}






