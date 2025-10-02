'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useAuthStore } from '@/store/auth'
import { useNotificationStore } from '@/store/notifications'
import { notificationService, Notification } from '@/lib/api/notifications'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { formatDate } from '@/lib/utils'

export default function NotificationsPage() {
  const router = useRouter()
  const { user, isAuthenticated } = useAuthStore()
  const { notifications, setNotifications, setUnreadCount, markAsRead, removeNotification } = useNotificationStore()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadNotifications()
  }, [isAuthenticated, router, user])

  const loadNotifications = async () => {
    if (!user) return
    try {
      const result = await notificationService.getNotifications(user.userId, 1, 50)
      setNotifications(result.notifications)
      setUnreadCount(result.unread_count)
    } catch (error) {
      console.error('Failed to load notifications:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleMarkAsRead = async (notificationId: string) => {
    if (!user) return
    try {
      await notificationService.markAsRead(notificationId, user.userId)
      markAsRead(notificationId)
    } catch (error) {
      console.error('Failed to mark as read:', error)
    }
  }

  const handleMarkAllAsRead = async () => {
    if (!user) return
    try {
      await notificationService.markAllAsRead(user.userId)
      await loadNotifications()
    } catch (error) {
      console.error('Failed to mark all as read:', error)
    }
  }

  const handleDelete = async (notificationId: string) => {
    if (!user) return
    try {
      await notificationService.deleteNotification(notificationId, user.userId)
      removeNotification(notificationId)
    } catch (error) {
      console.error('Failed to delete notification:', error)
    }
  }

  if (loading) {
    return <div className="flex min-h-screen items-center justify-center">Loading...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="border-b bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
          <h1 className="text-2xl font-bold">Notifications</h1>
          <div className="flex items-center gap-4">
            <Link href="/dashboard">
              <Button variant="outline">Back to Dashboard</Button>
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-8">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>All Notifications</CardTitle>
                <CardDescription>{notifications.length} notifications</CardDescription>
              </div>
              {notifications.some((n) => !n.is_read) && (
                <Button onClick={handleMarkAllAsRead}>Mark All as Read</Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {notifications.length === 0 ? (
                <p className="text-center text-gray-500 py-8">No notifications yet</p>
              ) : (
                notifications.map((notification) => (
                  <div
                    key={notification.notification_id}
                    className={`flex items-start justify-between rounded-lg border p-4 ${
                      !notification.is_read ? 'bg-blue-50 border-blue-200' : ''
                    }`}
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-medium">{notification.title}</h3>
                        {!notification.is_read && (
                          <span className="rounded-full bg-blue-500 px-2 py-1 text-xs text-white">
                            New
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-gray-600 mt-1">{notification.body}</p>
                      <p className="text-xs text-gray-400 mt-2">
                        {formatDate(notification.created_at)}
                      </p>
                      {notification.link && (
                        <Link
                          href={notification.link}
                          className="text-sm text-primary hover:underline mt-2 inline-block"
                        >
                          View →
                        </Link>
                      )}
                    </div>
                    <div className="flex gap-2 ml-4">
                      {!notification.is_read && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleMarkAsRead(notification.notification_id)}
                        >
                          Mark Read
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => handleDelete(notification.notification_id)}
                      >
                        ✕
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}

