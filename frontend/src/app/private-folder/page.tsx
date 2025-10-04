'use client'

import { useState, useEffect } from 'react'
import { useAuthStore } from '@/store/auth'
import { PrivateFolderView } from '@/components/PrivateFolderView'
import { PrivateFolderSettings } from '@/components/PrivateFolderSettings'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Lock, Settings } from 'lucide-react'

export default function PrivateFolderPage() {
  const { user, isAuthenticated } = useAuthStore()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) {
    return null
  }

  if (!isAuthenticated()) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="h-5 w-5" />
              Access Denied
            </CardTitle>
            <CardDescription>
              Please sign in to access your private folder
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              You need to be authenticated to access the private folder feature.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto px-4 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold mb-2">Private Folder</h1>
          <p className="text-muted-foreground">
            Secure your sensitive files with PIN protection
          </p>
        </div>

        <Tabs defaultValue="files" className="space-y-6">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="files" className="flex items-center gap-2">
              <Lock className="h-4 w-4" />
              Private Files
            </TabsTrigger>
            <TabsTrigger value="settings" className="flex items-center gap-2">
              <Settings className="h-4 w-4" />
              Settings
            </TabsTrigger>
          </TabsList>

          <TabsContent value="files">
            <PrivateFolderView userId={user?.userId || ''} />
          </TabsContent>

          <TabsContent value="settings">
            <PrivateFolderSettings userId={user?.userId || ''} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
