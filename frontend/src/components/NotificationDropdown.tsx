'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Bell,
  Check,
  X,
  Upload,
  Download,
  Share2,
  Trash2,
  AlertCircle,
  Clock
} from 'lucide-react'
import { useNotificationStore } from '@/store/notifications'
import { Notification } from '@/lib/api/notifications'

interface NotificationDropdownProps {
  onClose: () => void
}

const notificationIcons: Record<string, JSX.Element> = {
  upload: <Upload className="w-4 h-4 text-blue-500" />,
  download: <Download className="w-4 h-4 text-green-500" />,
  share: <Share2 className="w-4 h-4 text-purple-500" />,
  delete: <Trash2 className="w-4 h-4 text-red-500" />,
  error: <AlertCircle className="w-4 h-4 text-destructive" />,
  info: <Bell className="w-4 h-4 text-blue-500" />,
  file_shared: <Share2 className="w-4 h-4 text-purple-500" />,
  file_uploaded: <Upload className="w-4 h-4 text-blue-500" />,
  file_downloaded: <Download className="w-4 h-4 text-green-500" />,
  file_deleted: <Trash2 className="w-4 h-4 text-red-500" />
}

const notificationColors: Record<string, string> = {
  upload: 'bg-blue-50 text-blue-700 border-blue-200',
  download: 'bg-green-50 text-green-700 border-green-200',
  share: 'bg-purple-50 text-purple-700 border-purple-200',
  delete: 'bg-red-50 text-red-700 border-red-200',
  error: 'bg-destructive/10 text-destructive border-destructive/20',
  info: 'bg-blue-50 text-blue-700 border-blue-200',
  file_shared: 'bg-purple-50 text-purple-700 border-purple-200',
  file_uploaded: 'bg-blue-50 text-blue-700 border-blue-200',
  file_downloaded: 'bg-green-50 text-green-700 border-green-200',
  file_deleted: 'bg-red-50 text-red-700 border-red-200'
}

export function NotificationDropdown({ onClose }: NotificationDropdownProps) {
  const { notifications, unreadCount, markAsRead, markAllAsRead, clearAll } = useNotificationStore()
  const [localNotifications, setLocalNotifications] = useState<Notification[]>([])

  useEffect(() => {
    setLocalNotifications(notifications)
  }, [notifications])

  const formatTime = (timestamp: string) => {
    if (!timestamp) return 'Unknown time'
    
    const date = new Date(timestamp)
    
    // Check if the date is valid
    if (isNaN(date.getTime())) {
      return 'Unknown time'
    }
    
    const now = new Date()
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60))
    
    if (diffInMinutes < 1) return 'Just now'
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`
    return date.toLocaleDateString()
  }

  const handleMarkAsRead = (id: string) => {
    markAsRead(id)
    setLocalNotifications(prev =>
      prev.map(notif =>
        notif.notification_id === id ? { ...notif, is_read: true } : notif
      )
    )
  }

  const handleMarkAllAsRead = () => {
    markAllAsRead()
    setLocalNotifications(prev => 
      prev.map(notif => ({ ...notif, is_read: true }))
    )
  }

  const handleClearAll = () => {
    clearAll()
    setLocalNotifications([])
  }

  const unreadNotifications = localNotifications.filter(n => !n.is_read)
  const recentNotifications = localNotifications.slice(0, 10)

  return (
    <div className="absolute right-0 top-full mt-2 w-80 z-50">
      <Card className="shadow-lg border-0">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg font-semibold">Notifications</CardTitle>
            <div className="flex items-center space-x-2">
              {unreadCount > 0 && (
                <Badge variant="destructive" className="text-xs">
                  {unreadCount}
                </Badge>
              )}
              <Button
                variant="ghost"
                size="icon"
                onClick={onClose}
                className="h-6 w-6"
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>
          {unreadCount > 0 && (
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleMarkAllAsRead}
                className="text-xs"
              >
                <Check className="w-3 h-3 mr-1" />
                Mark all as read
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClearAll}
                className="text-xs text-destructive hover:text-destructive"
              >
                Clear all
              </Button>
            </div>
          )}
        </CardHeader>

        <CardContent className="p-0">
          {recentNotifications.length === 0 ? (
            <div className="text-center py-8">
              <Bell className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
              <p className="text-sm text-muted-foreground">No notifications yet</p>
            </div>
          ) : (
            <div className="max-h-96 overflow-y-auto">
              {recentNotifications.map((notification) => (
                <div
                  key={notification.notification_id}
                  className={`p-4 border-b border-border last:border-b-0 hover:bg-muted/50 transition-colors ${
                    !notification.is_read ? 'bg-muted/30' : ''
                  }`}
                  onClick={() => !notification.is_read && handleMarkAsRead(notification.notification_id)}
                >
                  <div className="flex items-start space-x-3">
                    <div className={`p-2 rounded-full border ${
                      notificationColors[notification.type] || notificationColors.info
                    }`}>
                      {notificationIcons[notification.type] || notificationIcons.info}
                    </div>

                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-foreground">
                        {notification.body}
                      </p>
                      <div className="flex items-center space-x-2 mt-1">
                        <Clock className="w-3 h-3 text-muted-foreground" />
                        <span className="text-xs text-muted-foreground">
                          {formatTime(notification.created_at)}
                        </span>
                        {!notification.is_read && (
                          <div className="w-2 h-2 bg-primary rounded-full" />        
                        )}
                      </div>
                    </div>

                    {!notification.is_read && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={(e) => {
                          e.stopPropagation()
                          handleMarkAsRead(notification.notification_id)
                        }}
                        className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <Check className="w-3 h-3" />
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>

        {recentNotifications.length > 0 && (
          <div className="p-3 border-t border-border">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {/* Navigate to full notifications page */}}
              className="w-full"
            >
              View all notifications
            </Button>
          </div>
        )}
      </Card>
    </div>
  )
}

