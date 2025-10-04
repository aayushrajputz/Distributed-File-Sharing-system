'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { authService, User } from '@/lib/api/auth'
import { useTheme } from '@/contexts/ThemeContext'
import { useLanguage } from '@/contexts/LanguageContext'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { 
  ArrowLeft,
  User as UserIcon,
  Bell,
  Shield,
  Palette,
  Globe,
  Save,
  Eye,
  EyeOff,
  Upload,
  Trash2,
  AlertTriangle
} from 'lucide-react'

export default function SettingsPage() {
  const router = useRouter()
  const { user, accessToken, refreshToken, isAuthenticated, setAuth, clearAuth } = useAuthStore()
  const { theme, setTheme } = useTheme()
  const { language, setLanguage, t } = useLanguage()
  const [profile, setProfile] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [mounted, setMounted] = useState(false)
  const [showCurrentPassword, setShowCurrentPassword] = useState(false)
  const [showNewPassword, setShowNewPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  
  // Profile settings
  const [profileData, setProfileData] = useState({
    full_name: '',
    email: '',
    avatar_url: ''
  })
  
  // Password change
  const [passwordData, setPasswordData] = useState({
    current_password: '',
    new_password: '',
    confirm_password: ''
  })
  
  // Notification settings
  const [notifications, setNotifications] = useState({
    email_notifications: true,
    file_shared: true,
    file_downloaded: true,
    system_updates: false,
    marketing_emails: false
  })
  
  // Privacy settings
  const [privacy, setPrivacy] = useState({
    profile_visibility: 'private',
    show_online_status: true,
    allow_file_sharing: true,
    data_retention: '1_year'
  })
  
  // Timezone setting (not implemented yet)
  const [timezone, setTimezone] = useState('UTC')

  useEffect(() => {
    setMounted(true)
  }, [])

  useEffect(() => {
    if (!mounted) return
    
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadProfile()
  }, [mounted, isAuthenticated, router, user])

  const loadProfile = async () => {
    if (!user) return
    
    try {
      const response = await authService.getUser(user.userId)
      setProfile(response.user)
      setProfileData({
        full_name: response.user.fullName,
        email: response.user.email,
        avatar_url: response.user.avatarUrl || ''
      })
    } catch (error) {
      console.error('Failed to load profile:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveProfile = async () => {
    setSaving(true)
    try {
      // Here you would typically call an API to update the profile
      // For now, we'll just simulate the update
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // Update the auth store with new profile data
      if (user && accessToken && refreshToken) {
        setAuth({
          ...user,
          fullName: profileData.full_name,
          email: profileData.email,
          avatarUrl: profileData.avatar_url
        }, accessToken, refreshToken)
      }
      
      alert('Profile updated successfully!')
    } catch (error) {
      console.error('Failed to update profile:', error)
      alert('Failed to update profile. Please try again.')
    } finally {
      setSaving(false)
    }
  }

  const handleChangePassword = async () => {
    if (!user) return
    
    if (passwordData.new_password !== passwordData.confirm_password) {
      alert('New passwords do not match!')
      return
    }
    
    if (passwordData.new_password.length < 8) {
      alert('New password must be at least 8 characters long!')
      return
    }
    
    setSaving(true)
    try {
      await authService.changePassword(
        user.userId,
        passwordData.current_password,
        passwordData.new_password
      )
      
      setPasswordData({
        current_password: '',
        new_password: '',
        confirm_password: ''
      })
      
      alert('Password changed successfully!')
    } catch (error: any) {
      console.error('Failed to change password:', error)
      const errorMessage = error.response?.data?.message || 'Failed to change password. Please try again.'
      alert(errorMessage)
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteAccount = () => {
    if (confirm('Are you sure you want to delete your account? This action cannot be undone.')) {
      // Here you would typically call an API to delete the account
      clearAuth()
      router.push('/')
    }
  }

  const handleSaveAppearance = () => {
    // Theme and language are automatically saved by their contexts
    // Only timezone needs manual saving
    alert(t('settings.saved'))
  }

  if (!mounted || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-indigo-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-indigo-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      {/* Header */}
      <header className="border-b bg-card/50 backdrop-blur-sm">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => router.push('/dashboard')}
                className="flex items-center space-x-2"
              >
                <ArrowLeft className="w-4 h-4" />
                <span>Back to Dashboard</span>
              </Button>
            </div>
            <div className="flex items-center space-x-4">
              <Button
                variant="outline"
                onClick={() => router.push('/profile')}
                className="flex items-center space-x-2"
              >
                <UserIcon className="w-4 h-4" />
                <span>Profile</span>
              </Button>
            </div>
          </div>
        </div>
      </header>

      <div className="container mx-auto px-4 py-8">
        <div className="max-w-4xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Settings</h1>
            <p className="text-gray-600 dark:text-gray-300 mt-2">
              Manage your account settings and preferences
            </p>
          </div>

          <Tabs defaultValue="profile" className="space-y-6">
            <TabsList className="grid w-full grid-cols-5">
              <TabsTrigger value="profile">Profile</TabsTrigger>
              <TabsTrigger value="security">Security</TabsTrigger>
              <TabsTrigger value="notifications">Notifications</TabsTrigger>
              <TabsTrigger value="privacy">Privacy</TabsTrigger>
              <TabsTrigger value="appearance">Appearance</TabsTrigger>
            </TabsList>

            {/* Profile Settings */}
            <TabsContent value="profile">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <UserIcon className="w-5 h-5" />
                    <span>Profile Information</span>
                  </CardTitle>
                  <CardDescription>
                    Update your personal information and profile picture
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center space-x-6">
                    <Avatar className="w-20 h-20">
                      <AvatarImage src={profileData.avatar_url || ''} alt={profileData.full_name} />
                      <AvatarFallback className="text-xl">
                        {profileData.full_name.split(' ').map(n => n[0]).join('')}
                      </AvatarFallback>
                    </Avatar>
                    <div className="space-y-2">
                      <Button variant="outline" size="sm">
                        <Upload className="w-4 h-4 mr-2" />
                        Upload Photo
                      </Button>
                      <p className="text-sm text-gray-500">
                        JPG, PNG or GIF. Max size 2MB.
                      </p>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="full_name">Full Name</Label>
                      <Input
                        id="full_name"
                        value={profileData.full_name}
                        onChange={(e) => setProfileData({...profileData, full_name: e.target.value})}
                        placeholder="Enter your full name"
                        autoComplete="name"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="email">Email</Label>
                      <Input
                        id="email"
                        type="email"
                        value={profileData.email}
                        onChange={(e) => setProfileData({...profileData, email: e.target.value})}
                        placeholder="Enter your email"
                        autoComplete="email"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="avatar_url">Avatar URL</Label>
                    <Input
                      id="avatar_url"
                      value={profileData.avatar_url}
                      onChange={(e) => setProfileData({...profileData, avatar_url: e.target.value})}
                      placeholder="Enter avatar URL"
                    />
                  </div>

                  <Button onClick={handleSaveProfile} disabled={saving} className="w-full">
                    <Save className="w-4 h-4 mr-2" />
                    {saving ? 'Saving...' : 'Save Changes'}
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Security Settings */}
            <TabsContent value="security">
              <div className="space-y-6">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Shield className="w-5 h-5" />
                      <span>Change Password</span>
                    </CardTitle>
                    <CardDescription>
                      Update your password to keep your account secure
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="current_password">Current Password</Label>
                      <div className="relative">
                        <Input
                          id="current_password"
                          type={showCurrentPassword ? "text" : "password"}
                          value={passwordData.current_password}
                          onChange={(e) => setPasswordData({...passwordData, current_password: e.target.value})}
                          placeholder="Enter current password"
                          autoComplete="current-password"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                          onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                        >
                          {showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="new_password">New Password</Label>
                      <div className="relative">
                        <Input
                          id="new_password"
                          type={showNewPassword ? "text" : "password"}
                          value={passwordData.new_password}
                          onChange={(e) => setPasswordData({...passwordData, new_password: e.target.value})}
                          placeholder="Enter new password"
                          autoComplete="new-password"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                          onClick={() => setShowNewPassword(!showNewPassword)}
                        >
                          {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="confirm_password">Confirm New Password</Label>
                      <div className="relative">
                        <Input
                          id="confirm_password"
                          type={showConfirmPassword ? "text" : "password"}
                          value={passwordData.confirm_password}
                          onChange={(e) => setPasswordData({...passwordData, confirm_password: e.target.value})}
                          placeholder="Confirm new password"
                          autoComplete="new-password"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                          onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                        >
                          {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </Button>
                      </div>
                    </div>

                    <Button onClick={handleChangePassword} disabled={saving} className="w-full">
                      <Shield className="w-4 h-4 mr-2" />
                      {saving ? 'Updating...' : 'Update Password'}
                    </Button>
                  </CardContent>
                </Card>

                <Card className="border-red-200 dark:border-red-800">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2 text-red-600">
                      <AlertTriangle className="w-5 h-5" />
                      <span>Danger Zone</span>
                    </CardTitle>
                    <CardDescription>
                      Permanently delete your account and all associated data
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <Button 
                      variant="destructive" 
                      onClick={handleDeleteAccount}
                      className="flex items-center space-x-2"
                    >
                      <Trash2 className="w-4 h-4" />
                      <span>Delete Account</span>
                    </Button>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            {/* Notification Settings */}
            <TabsContent value="notifications">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Bell className="w-5 h-5" />
                    <span>Notification Preferences</span>
                  </CardTitle>
                  <CardDescription>
                    Choose how you want to be notified about activities
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>Email Notifications</Label>
                        <p className="text-sm text-gray-500">
                          Receive notifications via email
                        </p>
                      </div>
                      <Switch
                        checked={notifications.email_notifications}
                        onCheckedChange={(checked) => 
                          setNotifications({...notifications, email_notifications: checked})
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>File Shared</Label>
                        <p className="text-sm text-gray-500">
                          Notify when someone shares a file with you
                        </p>
                      </div>
                      <Switch
                        checked={notifications.file_shared}
                        onCheckedChange={(checked) => 
                          setNotifications({...notifications, file_shared: checked})
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>File Downloaded</Label>
                        <p className="text-sm text-gray-500">
                          Notify when someone downloads your shared file
                        </p>
                      </div>
                      <Switch
                        checked={notifications.file_downloaded}
                        onCheckedChange={(checked) => 
                          setNotifications({...notifications, file_downloaded: checked})
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>System Updates</Label>
                        <p className="text-sm text-gray-500">
                          Notify about system updates and maintenance
                        </p>
                      </div>
                      <Switch
                        checked={notifications.system_updates}
                        onCheckedChange={(checked) => 
                          setNotifications({...notifications, system_updates: checked})
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>Marketing Emails</Label>
                        <p className="text-sm text-gray-500">
                          Receive promotional emails and updates
                        </p>
                      </div>
                      <Switch
                        checked={notifications.marketing_emails}
                        onCheckedChange={(checked) => 
                          setNotifications({...notifications, marketing_emails: checked})
                        }
                      />
                    </div>
                  </div>

                  <Button onClick={() => alert('Notification settings saved!')} className="w-full">
                    <Save className="w-4 h-4 mr-2" />
                    Save Notification Settings
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Privacy Settings */}
            <TabsContent value="privacy">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Globe className="w-5 h-5" />
                    <span>Privacy Settings</span>
                  </CardTitle>
                  <CardDescription>
                    Control your privacy and data sharing preferences
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label>Profile Visibility</Label>
                      <Select 
                        value={privacy.profile_visibility}
                        onValueChange={(value) => setPrivacy({...privacy, profile_visibility: value})}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="public">Public</SelectItem>
                          <SelectItem value="private">Private</SelectItem>
                          <SelectItem value="friends">Friends Only</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>Show Online Status</Label>
                        <p className="text-sm text-gray-500">
                          Let others see when you&apos;re online
                        </p>
                      </div>
                      <Switch
                        checked={privacy.show_online_status}
                        onCheckedChange={(checked) => 
                          setPrivacy({...privacy, show_online_status: checked})
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>Allow File Sharing</Label>
                        <p className="text-sm text-gray-500">
                          Allow others to share files with you
                        </p>
                      </div>
                      <Switch
                        checked={privacy.allow_file_sharing}
                        onCheckedChange={(checked) => 
                          setPrivacy({...privacy, allow_file_sharing: checked})
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <Label>Data Retention</Label>
                      <Select 
                        value={privacy.data_retention}
                        onValueChange={(value) => setPrivacy({...privacy, data_retention: value})}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="30_days">30 Days</SelectItem>
                          <SelectItem value="1_year">1 Year</SelectItem>
                          <SelectItem value="5_years">5 Years</SelectItem>
                          <SelectItem value="forever">Forever</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <Button onClick={() => alert('Privacy settings saved!')} className="w-full">
                    <Save className="w-4 h-4 mr-2" />
                    Save Privacy Settings
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            {/* Appearance Settings */}
            <TabsContent value="appearance">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center space-x-2">
                    <Palette className="w-5 h-5" />
                    <span>{t('settings.appearance')} & {t('settings.language')}</span>
                  </CardTitle>
                  <CardDescription>
                    Customize the look and feel of your interface
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label>{t('settings.theme')}</Label>
                      <Select 
                        value={theme}
                        onValueChange={(value) => setTheme(value as 'light' | 'dark' | 'system')}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="light">{t('settings.light')}</SelectItem>
                          <SelectItem value="dark">{t('settings.dark')}</SelectItem>
                          <SelectItem value="system">{t('settings.system')}</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label>{t('settings.language')}</Label>
                      <Select 
                        value={language}
                        onValueChange={(value) => setLanguage(value as 'en' | 'es' | 'fr' | 'de')}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="en">{t('settings.english')}</SelectItem>
                          <SelectItem value="es">{t('settings.spanish')}</SelectItem>
                          <SelectItem value="fr">{t('settings.french')}</SelectItem>
                          <SelectItem value="de">{t('settings.german')}</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label>{t('settings.timezone')}</Label>
                      <Select 
                        value={timezone}
                        onValueChange={(value) => setTimezone(value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="UTC">{t('settings.utc')}</SelectItem>
                          <SelectItem value="EST">{t('settings.est')}</SelectItem>
                          <SelectItem value="PST">{t('settings.pst')}</SelectItem>
                          <SelectItem value="GMT">{t('settings.gmt')}</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <Button onClick={handleSaveAppearance} className="w-full">
                    <Save className="w-4 h-4 mr-2" />
                    {t('settings.save')} {t('settings.appearance')} Settings
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  )
}
