'use client'

import { useRouter, usePathname } from 'next/navigation'
import { 
  LayoutDashboard, 
  FolderOpen, 
  Users, 
  Star, 
  Settings, 
  ChevronLeft,
  ChevronRight
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface SidebarProps {
  collapsed?: boolean
  onToggle?: () => void
  userId?: string
}

export function Sidebar({ collapsed = false, onToggle, userId }: SidebarProps) {
  const router = useRouter()
  const pathname = usePathname()

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
    { name: 'My Files', href: '/dashboard', icon: FolderOpen },
    { name: 'Shared with Me', href: '/dashboard/shared', icon: Users },
    { name: 'Favorites', href: '/dashboard/favorites', icon: Star },
    { name: 'Settings', href: '/settings', icon: Settings },
  ]

  return (
    <aside
      className={cn(
        'fixed left-0 top-0 z-40 h-screen border-r border-border/40 bg-gradient-to-b from-background via-background to-muted/20 backdrop-blur-xl transition-all duration-300',
        collapsed ? 'w-20' : 'w-64'
      )}
    >
      <div className="flex h-full flex-col">
        {/* Logo */}
        <div className="flex h-16 items-center justify-between border-b border-border/40 px-4">
          {!collapsed && (
            <div className="flex items-center space-x-2">
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-xl blur-md opacity-50"></div>
                <div className="relative w-10 h-10 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-xl flex items-center justify-center shadow-lg">
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
                  </svg>
                </div>
              </div>
              <div>
                <h1 className="text-lg font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                  CloudShare
                </h1>
                <p className="text-xs text-muted-foreground">Pro Plan</p>
              </div>
            </div>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggle}
            className="h-8 w-8 hover:bg-muted/50"
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </Button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 px-3 py-4">
          {navigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <Button
                key={item.name}
                variant={isActive ? 'secondary' : 'ghost'}
                className={cn(
                  'w-full justify-start transition-all duration-200',
                  collapsed ? 'px-2' : 'px-3',
                  isActive && 'bg-gradient-to-r from-blue-600/10 to-indigo-600/10 border border-blue-600/20 shadow-sm'
                )}
                onClick={() => router.push(item.href)}
              >
                <item.icon className={cn('h-5 w-5', isActive && 'text-blue-600')} />
                {!collapsed && (
                  <span className={cn('ml-3', isActive && 'font-semibold')}>{item.name}</span>
                )}
              </Button>
            )
          })}
        </nav>


      </div>
    </aside>
  )
}

