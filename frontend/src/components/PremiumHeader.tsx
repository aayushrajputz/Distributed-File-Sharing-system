'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { 
  Search, 
  Bell, 
  Moon, 
  Sun, 
  Settings, 
  LogOut,
  User,
  CreditCard,
  HelpCircle,
  ChevronDown
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { NotificationDropdown } from '@/components/NotificationDropdown'
import { useAuthStore } from '@/store/auth'
import { useNotificationStore } from '@/store/notifications'
import { cn } from '@/lib/utils'

interface PremiumHeaderProps {
  searchQuery: string
  onSearchChange: (query: string) => void
  darkMode?: boolean
  onToggleDarkMode?: () => void
  sidebarCollapsed?: boolean
}

export function PremiumHeader({ 
  searchQuery, 
  onSearchChange, 
  darkMode, 
  onToggleDarkMode,
  sidebarCollapsed = false
}: PremiumHeaderProps) {
  const router = useRouter()
  const { user, clearAuth } = useAuthStore()
  const { unreadCount } = useNotificationStore()
  const [showNotifications, setShowNotifications] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)

  const handleLogout = () => {
    clearAuth()
    router.push('/auth/login')
  }

  return (
    <header 
      className={cn(
        'fixed top-0 right-0 z-30 h-16 border-b border-border/40 bg-background/80 backdrop-blur-xl transition-all duration-300',
        sidebarCollapsed ? 'left-20' : 'left-64'
      )}
    >
      <div className="flex h-full items-center justify-between px-6">
        {/* Search Bar */}
        <div className="flex-1 max-w-2xl">
          <div className="relative group">
            <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 text-muted-foreground w-5 h-5 transition-colors group-focus-within:text-blue-600" />
            <Input
              placeholder="Search files, folders, and shared content..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-12 pr-4 h-11 bg-muted/50 border-border/40 rounded-xl focus:bg-background focus:border-blue-600/50 focus:ring-2 focus:ring-blue-600/20 transition-all"
              autoComplete="off"
            />
            <kbd className="absolute right-4 top-1/2 transform -translate-y-1/2 pointer-events-none hidden sm:inline-flex h-6 select-none items-center gap-1 rounded border border-border/40 bg-muted px-2 font-mono text-xs text-muted-foreground opacity-100">
              <span className="text-xs">⌘</span>K
            </kbd>
          </div>
        </div>

        {/* Right Actions */}
        <div className="flex items-center space-x-2 ml-4">
          {/* Notifications */}
          <div className="relative">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowNotifications(!showNotifications)}
              className="relative h-10 w-10 rounded-xl hover:bg-muted/50 transition-all"
            >
              <Bell className="w-5 h-5" />
              {unreadCount > 0 && (
                <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-r from-red-500 to-pink-500 text-xs font-bold text-white shadow-lg animate-pulse">
                  {unreadCount > 9 ? '9+' : unreadCount}
                </span>
              )}
            </Button>
            {showNotifications && (
              <div className="absolute right-0 mt-2">
                <NotificationDropdown onClose={() => setShowNotifications(false)} />
              </div>
            )}
          </div>

          {/* Dark Mode Toggle */}
          {onToggleDarkMode && (
            <Button
              variant="ghost"
              size="icon"
              onClick={onToggleDarkMode}
              className="h-10 w-10 rounded-xl hover:bg-muted/50 transition-all"
            >
              {darkMode ? (
                <Sun className="w-5 h-5 text-yellow-500" />
              ) : (
                <Moon className="w-5 h-5 text-blue-600" />
              )}
            </Button>
          )}

          {/* Divider */}
          <div className="h-8 w-px bg-border/40" />

          {/* User Menu */}
          <div className="relative">
            <Button
              variant="ghost"
              onClick={() => setShowUserMenu(!showUserMenu)}
              className="flex items-center space-x-3 h-10 px-3 rounded-xl hover:bg-muted/50 transition-all"
            >
              <Avatar className="h-8 w-8 ring-2 ring-blue-600/20">
                <AvatarImage src={user?.avatarUrl || ''} alt={user?.fullName || 'User'} />
                <AvatarFallback className="bg-gradient-to-r from-blue-600 to-indigo-600 text-white text-sm font-semibold">
                  {user?.fullName?.charAt(0) || 'U'}
                </AvatarFallback>
              </Avatar>
              <div className="hidden md:block text-left">
                <p className="text-sm font-semibold leading-none">{user?.fullName || 'User'}</p>
                <p className="text-xs text-muted-foreground mt-0.5">{user?.email || ''}</p>
              </div>
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            </Button>

            {/* User Dropdown Menu */}
            {showUserMenu && (
              <div className="absolute right-0 mt-2 w-64 rounded-xl border border-border/40 bg-background/95 backdrop-blur-xl shadow-2xl overflow-hidden animate-in fade-in slide-in-from-top-2 duration-200">
                {/* User Info */}
                <div className="p-4 border-b border-border/40 bg-gradient-to-r from-blue-600/10 to-indigo-600/10">
                  <div className="flex items-center space-x-3">
                    <Avatar className="h-12 w-12 ring-2 ring-blue-600/20">
                      <AvatarImage src={user?.avatarUrl || ''} alt={user?.fullName || 'User'} />
                      <AvatarFallback className="bg-gradient-to-r from-blue-600 to-indigo-600 text-white font-semibold">
                        {user?.fullName?.charAt(0) || 'U'}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold truncate">{user?.fullName || 'User'}</p>
                      <p className="text-xs text-muted-foreground truncate">{user?.email || ''}</p>
                      <div className="mt-1">
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gradient-to-r from-blue-600 to-indigo-600 text-white">
                          Pro Plan
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Menu Items */}
                <div className="p-2">
                  <Button
                    variant="ghost"
                    className="w-full justify-start h-10 px-3 rounded-lg hover:bg-muted/50"
                    onClick={() => {
                      router.push('/profile')
                      setShowUserMenu(false)
                    }}
                  >
                    <User className="w-4 h-4 mr-3" />
                    <span>My Profile</span>
                  </Button>
                  
                  <Button
                    variant="ghost"
                    className="w-full justify-start h-10 px-3 rounded-lg hover:bg-muted/50"
                    onClick={() => {
                      router.push('/settings')
                      setShowUserMenu(false)
                    }}
                  >
                    <Settings className="w-4 h-4 mr-3" />
                    <span>Settings</span>
                  </Button>

                  <Button
                    variant="ghost"
                    className="w-full justify-start h-10 px-3 rounded-lg hover:bg-muted/50"
                    onClick={() => {
                      router.push('/billing')
                      setShowUserMenu(false)
                    }}
                  >
                    <CreditCard className="w-4 h-4 mr-3" />
                    <span>Billing</span>
                  </Button>

                  <Button
                    variant="ghost"
                    className="w-full justify-start h-10 px-3 rounded-lg hover:bg-muted/50"
                    onClick={() => {
                      router.push('/help')
                      setShowUserMenu(false)
                    }}
                  >
                    <HelpCircle className="w-4 h-4 mr-3" />
                    <span>Help & Support</span>
                  </Button>
                </div>

                {/* Logout */}
                <div className="p-2 border-t border-border/40">
                  <Button
                    variant="ghost"
                    className="w-full justify-start h-10 px-3 rounded-lg hover:bg-red-500/10 hover:text-red-500 transition-colors"
                    onClick={() => {
                      handleLogout()
                      setShowUserMenu(false)
                    }}
                  >
                    <LogOut className="w-4 h-4 mr-3" />
                    <span>Log Out</span>
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  )
}

