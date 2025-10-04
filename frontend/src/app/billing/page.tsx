'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { billingService, Plan, Subscription, Usage } from '@/lib/api/billing'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Sidebar } from '@/components/Sidebar'
import { PremiumHeader } from '@/components/PremiumHeader'
import { 
  CreditCard, 
  Check, 
  X, 
  AlertCircle, 
  HardDrive, 
  TrendingUp,
  Crown,
  Zap,
  Building
} from 'lucide-react'
import { cn } from '@/lib/utils'

export default function BillingPage() {
  const router = useRouter()
  const { user, isAuthenticated } = useAuthStore()
  const [mounted, setMounted] = useState(false)
  const [loading, setLoading] = useState(true)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  
  // Data states
  const [plans, setPlans] = useState<Plan[]>([])
  const [subscription, setSubscription] = useState<Subscription | null>(null)
  const [usage, setUsage] = useState<Usage | null>(null)
  const [hasActiveSubscription, setHasActiveSubscription] = useState(false)

  const loadBillingData = useCallback(async () => {
    if (!user) return
    
    try {
      setLoading(true)
      
      // Mock professional data for now
      const mockPlans = [
        {
          id: 'free',
          name: 'Free',
          description: 'Perfect for personal use',
          price_per_month: 0,
          quota_bytes: 5 * 1024 * 1024 * 1024, // 5GB
          features: [
            '5GB Storage',
            'Basic file sharing',
            'Email support',
            'Standard security'
          ],
          is_popular: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        },
        {
          id: 'pro',
          name: 'Pro',
          description: 'Best for professionals and small teams',
          price_per_month: 9.99,
          quota_bytes: 100 * 1024 * 1024 * 1024, // 100GB
          features: [
            '100GB Storage',
            'Advanced file sharing',
            'Priority support',
            'Enhanced security',
            'Custom branding',
            'API access'
          ],
          is_popular: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        },
        {
          id: 'enterprise',
          name: 'Enterprise',
          description: 'For large organizations',
          price_per_month: 29.99,
          quota_bytes: 1000 * 1024 * 1024 * 1024, // 1TB
          features: [
            '1TB Storage',
            'Unlimited file sharing',
            '24/7 phone support',
            'Enterprise security',
            'White-label solution',
            'Advanced analytics',
            'SSO integration'
          ],
          is_popular: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        }
      ]
      
      // Try to get real usage data, fallback to mock
      let usageData = null
      try {
        const usageResponse = await billingService.getUsage(user.userId)
        usageData = usageResponse.usage
      } catch (error) {
        console.warn('Using mock usage data:', error)
        // Calculate mock usage based on uploaded files
        const mockUsedBytes = 2.5 * 1024 * 1024 * 1024 // 2.5GB
        usageData = {
          user_id: user.userId,
          plan_name: 'Free',
          quota_bytes: 5 * 1024 * 1024 * 1024,
          used_bytes: mockUsedBytes,
          quota_gb: 5,
          used_gb: 2.5,
          percent_used: 50,
          upgrade_available: true,
          quota_exceeded: false
        }
      }
      
      setPlans(mockPlans)
      setSubscription(null) // No active subscription for now
      setHasActiveSubscription(false)
      setUsage(usageData)
    } catch (error) {
      console.error('Failed to load billing data:', error)
    } finally {
      setLoading(false)
    }
  }, [user])

  useEffect(() => {
    setMounted(true)
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadBillingData()
  }, [isAuthenticated, router, user, loadBillingData])

  const handleUpgrade = async (planId: string) => {
    if (!user) return
    
    try {
      const response = await billingService.createSubscription({
        user_id: user.userId,
        plan_id: planId,
        payment_method: 'stripe'
      })
      
      // Redirect to Stripe checkout
      window.location.href = response.payment_url
    } catch (error) {
      console.error('Failed to create subscription:', error)
    }
  }

  const handleCancelSubscription = async () => {
    if (!subscription || !user) return
    
    try {
      await billingService.cancelSubscription({
        user_id: user.userId,
        subscription_id: subscription.id
      })
      
      // Reload data
      loadBillingData()
    } catch (error) {
      console.error('Failed to cancel subscription:', error)
    }
  }

  const getPlanIcon = (planName: string) => {
    switch (planName.toLowerCase()) {
      case 'free':
        return <HardDrive className="h-6 w-6" />
      case 'pro':
        return <Zap className="h-6 w-6" />
      case 'enterprise':
        return <Building className="h-6 w-6" />
      default:
        return <Crown className="h-6 w-6" />
    }
  }

  const getPlanColor = (planName: string) => {
    switch (planName.toLowerCase()) {
      case 'free':
        return 'text-gray-600'
      case 'pro':
        return 'text-blue-600'
      case 'enterprise':
        return 'text-purple-600'
      default:
        return 'text-gray-600'
    }
  }

  if (!mounted) return null

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-100 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
      <Sidebar collapsed={sidebarCollapsed} onToggle={() => setSidebarCollapsed(!sidebarCollapsed)} />
      
      <div className={cn(
        "transition-all duration-300",
        sidebarCollapsed ? "ml-20" : "ml-64"
      )}>
        <PremiumHeader 
          searchQuery=""
          onSearchChange={() => {}}
          darkMode={false}
          onToggleDarkMode={() => {}}
          sidebarCollapsed={sidebarCollapsed}
        />
        
        <main className="p-6">
          <div className="max-w-7xl mx-auto space-y-8">
            {/* Header */}
            <div className="text-center space-y-6">
              <div className="space-y-4">
                <h1 className="text-5xl font-bold bg-gradient-to-r from-blue-600 via-purple-600 to-indigo-600 bg-clip-text text-transparent">
                  Billing & Storage
                </h1>
                <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
                  Manage your subscription, storage usage, and upgrade your plan to unlock more features
                </p>
              </div>
              
              {/* Quick Stats */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 max-w-4xl mx-auto">
                <div className="bg-gradient-to-r from-blue-50 to-blue-100 dark:from-blue-900/20 dark:to-blue-800/20 p-4 rounded-lg border border-blue-200 dark:border-blue-800">
                  <div className="text-2xl font-bold text-blue-600">17</div>
                  <div className="text-sm text-blue-600/80">Files Uploaded</div>
                </div>
                <div className="bg-gradient-to-r from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20 p-4 rounded-lg border border-green-200 dark:border-green-800">
                  <div className="text-2xl font-bold text-green-600">2.5 GB</div>
                  <div className="text-sm text-green-600/80">Storage Used</div>
                </div>
                <div className="bg-gradient-to-r from-purple-50 to-purple-100 dark:from-purple-900/20 dark:to-purple-800/20 p-4 rounded-lg border border-purple-200 dark:border-purple-800">
                  <div className="text-2xl font-bold text-purple-600">Free</div>
                  <div className="text-sm text-purple-600/80">Current Plan</div>
                </div>
              </div>
            </div>

            {loading ? (
              <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              </div>
            ) : (
              <>
                {/* Current Plan & Usage */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {/* Current Plan */}
                  <Card className="border-2 border-blue-200 bg-gradient-to-br from-blue-50 to-indigo-50">
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <CreditCard className="h-5 w-5" />
                        Current Plan
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      {subscription ? (
                        <div className="space-y-3">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              {getPlanIcon(subscription.plan?.name || 'Free')}
                              <span className="text-xl font-semibold">
                                {subscription.plan?.name || 'Free'}
                              </span>
                            </div>
                            <Badge 
                              variant={subscription.status === 'active' ? 'default' : 'secondary'}
                              className={cn(
                                subscription.status === 'active' ? 'bg-green-100 text-green-800' : ''
                              )}
                            >
                              {subscription.status}
                            </Badge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            ${subscription.plan?.price_per_month || 0}/month
                          </p>
                          {subscription.status === 'active' && (
                            <Button 
                              variant="outline" 
                              size="sm"
                              onClick={handleCancelSubscription}
                              className="text-red-600 border-red-200 hover:bg-red-50"
                            >
                              Cancel Subscription
                            </Button>
                          )}
                        </div>
                      ) : (
                        <div className="text-center py-4">
                          <p className="text-muted-foreground">No active subscription</p>
                          <p className="text-sm text-muted-foreground">You&apos;re on the Free plan</p>
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  {/* Storage Usage */}
                  <Card className="border-2 border-green-200 bg-gradient-to-br from-green-50 to-emerald-50">
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <HardDrive className="h-5 w-5" />
                        Storage Usage
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      {usage ? (
                        <div className="space-y-4">
                          <div className="flex items-center justify-between">
                            <div>
                              <span className="text-3xl font-bold text-green-600">
                                {usage.used_gb.toFixed(1)} GB
                              </span>
                              <span className="text-lg text-muted-foreground ml-2">
                                of {usage.quota_gb} GB
                              </span>
                            </div>
                            <div className="text-right">
                              <div className="text-sm text-muted-foreground">Files</div>
                              <div className="text-lg font-semibold">17</div>
                            </div>
                          </div>
                          
                          <Progress 
                            value={usage.percent_used} 
                            className="h-4"
                          />
                          
                          <div className="flex items-center justify-between text-sm">
                            <span className={cn(
                              "font-semibold text-lg",
                              usage.percent_used > 90 ? "text-red-600" : 
                              usage.percent_used > 75 ? "text-yellow-600" : "text-green-600"
                            )}>
                              {usage.percent_used.toFixed(1)}% used
                            </span>
                            <span className="text-muted-foreground font-medium">
                              {(usage.quota_gb - usage.used_gb).toFixed(1)} GB available
                            </span>
                          </div>

                          {/* Storage Validation Warnings */}
                          {usage.percent_used > 90 && (
                            <div className="flex items-center gap-2 p-4 bg-red-50 border border-red-200 rounded-lg">
                              <AlertCircle className="h-5 w-5 text-red-600" />
                              <div>
                                <div className="text-sm font-semibold text-red-800">
                                  Storage Almost Full!
                                </div>
                                <div className="text-xs text-red-700">
                                  You&apos;re using {usage.percent_used.toFixed(1)}% of your storage. Upgrade now to avoid service interruption.
                                </div>
                              </div>
                            </div>
                          )}
                          
                          {usage.percent_used > 75 && usage.percent_used <= 90 && (
                            <div className="flex items-center gap-2 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                              <AlertCircle className="h-5 w-5 text-yellow-600" />
                              <div>
                                <div className="text-sm font-semibold text-yellow-800">
                                  Storage Getting Full
                                </div>
                                <div className="text-xs text-yellow-700">
                                  Consider upgrading your plan or renting additional storage.
                                </div>
                              </div>
                            </div>
                          )}

                          {usage.quota_exceeded && (
                            <div className="flex items-center gap-2 p-4 bg-red-50 border border-red-200 rounded-lg">
                              <AlertCircle className="h-5 w-5 text-red-600" />
                              <div>
                                <div className="text-sm font-semibold text-red-800">
                                  Storage Limit Exceeded!
                                </div>
                                <div className="text-xs text-red-700">
                                  New uploads are blocked. Please upgrade your plan immediately.
                                </div>
                              </div>
                            </div>
                          )}

                          {/* Quick Actions */}
                          <div className="flex gap-2 pt-2">
                            <Button size="sm" variant="outline" className="flex-1">
                              <TrendingUp className="h-4 w-4 mr-1" />
                              Upgrade Plan
                            </Button>
                            <Button size="sm" variant="outline" className="flex-1">
                              <HardDrive className="h-4 w-4 mr-1" />
                              Rent Storage
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="text-center py-8">
                          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-green-600 mx-auto mb-4"></div>
                          <p className="text-muted-foreground">Loading usage data...</p>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>

                {/* Storage Rental Options */}
                <div className="space-y-6">
                  <h2 className="text-2xl font-bold text-center">Additional Storage</h2>
                  <p className="text-center text-muted-foreground">
                    Need more storage? Rent additional space without changing your plan
                  </p>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <Card className="border-2 border-orange-200 bg-gradient-to-br from-orange-50 to-yellow-50">
                      <CardHeader className="text-center">
                        <CardTitle className="flex items-center justify-center gap-2">
                          <HardDrive className="h-6 w-6 text-orange-600" />
                          +50GB Storage
                        </CardTitle>
                        <CardDescription>Perfect for temporary needs</CardDescription>
                        <div className="mt-4">
                          <span className="text-3xl font-bold">$2.99</span>
                          <span className="text-muted-foreground">/month</span>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <Button className="w-full" variant="outline">
                          Rent Now
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-2 border-green-200 bg-gradient-to-br from-green-50 to-emerald-50">
                      <CardHeader className="text-center">
                        <CardTitle className="flex items-center justify-center gap-2">
                          <HardDrive className="h-6 w-6 text-green-600" />
                          +200GB Storage
                        </CardTitle>
                        <CardDescription>Great for growing businesses</CardDescription>
                        <div className="mt-4">
                          <span className="text-3xl font-bold">$9.99</span>
                          <span className="text-muted-foreground">/month</span>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <Button className="w-full" variant="outline">
                          Rent Now
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-2 border-purple-200 bg-gradient-to-br from-purple-50 to-indigo-50">
                      <CardHeader className="text-center">
                        <CardTitle className="flex items-center justify-center gap-2">
                          <HardDrive className="h-6 w-6 text-purple-600" />
                          +500GB Storage
                        </CardTitle>
                        <CardDescription>For heavy data users</CardDescription>
                        <div className="mt-4">
                          <span className="text-3xl font-bold">$19.99</span>
                          <span className="text-muted-foreground">/month</span>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <Button className="w-full" variant="outline">
                          Rent Now
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                {/* Available Plans */}
                <div className="space-y-6">
                  <h2 className="text-2xl font-bold text-center">Choose Your Plan</h2>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {plans.map((plan) => (
                      <Card 
                        key={plan.id} 
                        className={cn(
                          "relative transition-all duration-200 hover:shadow-lg",
                          plan.is_popular ? "border-2 border-blue-500 shadow-lg scale-105" : "border border-gray-200",
                          subscription?.plan_id === plan.id ? "ring-2 ring-green-500" : ""
                        )}
                      >
                        {plan.is_popular && (
                          <div className="absolute -top-3 left-1/2 transform -translate-x-1/2">
                            <Badge className="bg-blue-600 text-white px-3 py-1">
                              Most Popular
                            </Badge>
                          </div>
                        )}
                        
                        {subscription?.plan_id === plan.id && (
                          <div className="absolute -top-3 right-4">
                            <Badge className="bg-green-600 text-white px-3 py-1">
                              <Check className="h-3 w-3 mr-1" />
                              Current
                            </Badge>
                          </div>
                        )}

                        <CardHeader className="text-center pb-4">
                          <div className="flex justify-center mb-2">
                            <div className={cn(
                              "p-3 rounded-full bg-gray-100",
                              getPlanColor(plan.name)
                            )}>
                              {getPlanIcon(plan.name)}
                            </div>
                          </div>
                          <CardTitle className="text-xl">{plan.name}</CardTitle>
                          <CardDescription>{plan.description}</CardDescription>
                          <div className="mt-4">
                            <span className="text-4xl font-bold">${plan.price_per_month}</span>
                            <span className="text-muted-foreground">/month</span>
                          </div>
                        </CardHeader>

                        <CardContent className="space-y-4">
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <HardDrive className="h-4 w-4 text-blue-600" />
                              <span className="text-sm">
                                {(plan.quota_bytes / (1024 * 1024 * 1024)).toFixed(0)} GB Storage
                              </span>
                            </div>
                          </div>

                          <div className="space-y-2">
                            {plan.features.map((feature, index) => (
                              <div key={index} className="flex items-center gap-2">
                                <Check className="h-4 w-4 text-green-600" />
                                <span className="text-sm">{feature}</span>
                              </div>
                            ))}
                          </div>

                          <Button 
                            className="w-full"
                            variant={plan.is_popular ? "default" : "outline"}
                            disabled={subscription?.plan_id === plan.id}
                            onClick={() => handleUpgrade(plan.id)}
                          >
                            {subscription?.plan_id === plan.id ? (
                              <>
                                <Check className="h-4 w-4 mr-2" />
                                Current Plan
                              </>
                            ) : plan.price_per_month === 0 ? (
                              'Current Plan'
                            ) : (
                              'Upgrade Now'
                            )}
                          </Button>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </div>

                {/* Payment History */}
                {subscription && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <TrendingUp className="h-5 w-5" />
                        Payment History
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                          <div>
                            <p className="font-medium">{subscription.plan?.name} Plan</p>
                            <p className="text-sm text-muted-foreground">
                              {new Date(subscription.start_date).toLocaleDateString()}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="font-medium">${subscription.plan?.price_per_month}</p>
                            <Badge 
                              variant={subscription.payment_status === 'paid' ? 'default' : 'secondary'}
                              className={cn(
                                subscription.payment_status === 'paid' ? 'bg-green-100 text-green-800' : ''
                              )}
                            >
                              {subscription.payment_status}
                            </Badge>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </>
            )}
          </div>
        </main>
      </div>
    </div>
  )
}