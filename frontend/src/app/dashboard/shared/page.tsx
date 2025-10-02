'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Users, Download, Share2, Trash2, MoreVertical, Grid, List, Search } from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Sidebar } from '@/components/Sidebar'
import { PremiumHeader } from '@/components/PremiumHeader'
import { PremiumFileCard } from '@/components/PremiumFileCard'
import { FileMetadata } from '@/lib/api/files'
import { cn } from '@/lib/utils'

export default function SharedFilesPage() {
  const router = useRouter()
  const { user, isAuthenticated } = useAuthStore()
  const [mounted, setMounted] = useState(false)
  const [loading, setLoading] = useState(true)
  const [sharedFiles, setSharedFiles] = useState<FileMetadata[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  useEffect(() => {
    setMounted(true)
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadSharedFiles()
  }, [isAuthenticated, router])

  const loadSharedFiles = async () => {
    try {
      setLoading(true)
      // TODO: Replace with actual API call
      // const response = await fileService.getSharedFiles()
      // setSharedFiles(response.data)
      
      // Mock data for now
      setSharedFiles([])
    } catch (error) {
      console.error('Failed to load shared files:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async (fileId: string, fileName: string) => {
    // TODO: Implement download
    console.log('Download:', fileId, fileName)
  }

  const handleShare = (fileId: string) => {
    // TODO: Implement share
    console.log('Share:', fileId)
  }

  const handleDelete = async (fileId: string, fileName: string) => {
    // TODO: Implement delete
    console.log('Delete:', fileId, fileName)
  }

  const filteredFiles = sharedFiles.filter(file =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (!mounted || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20 flex items-center justify-center">
        <div className="text-center">
          <div className="relative w-16 h-16 mx-auto mb-6">
            <div className="absolute inset-0 bg-gradient-to-r from-green-600 to-emerald-600 rounded-full blur-xl opacity-50 animate-pulse"></div>
            <div className="relative w-16 h-16 bg-gradient-to-r from-green-600 to-emerald-600 rounded-full flex items-center justify-center animate-spin">
              <div className="w-12 h-12 bg-background rounded-full"></div>
            </div>
          </div>
          <p className="text-lg font-semibold mb-2">Loading shared files...</p>
          <p className="text-sm text-muted-foreground">Please wait</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
      <Sidebar 
        collapsed={sidebarCollapsed} 
        onToggle={() => setSidebarCollapsed(!sidebarCollapsed)} 
      />

      <PremiumHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        sidebarCollapsed={sidebarCollapsed}
      />

      <main className={cn(
        'pt-16 transition-all duration-300',
        sidebarCollapsed ? 'ml-20' : 'ml-64'
      )}>
        <div className="p-8 space-y-8">
          {/* Page Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-green-600 to-emerald-600 flex items-center justify-center">
                <Users className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-3xl font-bold bg-gradient-to-r from-green-600 to-emerald-600 bg-clip-text text-transparent">
                  Shared with Me
                </h1>
                <p className="text-muted-foreground">Files that others have shared with you</p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <div className="flex border border-border/40 rounded-xl bg-background/50 backdrop-blur-sm overflow-hidden">
                <Button
                  variant={viewMode === 'grid' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('grid')}
                  className={cn(
                    'rounded-none h-10 px-4',
                    viewMode === 'grid' && 'bg-gradient-to-r from-green-600 to-emerald-600 text-white'
                  )}
                >
                  <Grid className="w-4 h-4" />
                </Button>
                <Button
                  variant={viewMode === 'list' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('list')}
                  className={cn(
                    'rounded-none h-10 px-4',
                    viewMode === 'list' && 'bg-gradient-to-r from-green-600 to-emerald-600 text-white'
                  )}
                >
                  <List className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>

          {/* Files Display */}
          {filteredFiles.length === 0 ? (
            <Card className="border-border/40 bg-background/50 backdrop-blur-sm text-center py-16">
              <CardContent>
                <div className="relative w-24 h-24 mx-auto mb-6">
                  <div className="absolute inset-0 bg-gradient-to-r from-green-600/20 to-emerald-600/20 rounded-full blur-2xl"></div>
                  <div className="relative w-24 h-24 bg-gradient-to-r from-green-600/10 to-emerald-600/10 rounded-full flex items-center justify-center border-2 border-dashed border-green-600/30">
                    <Users className="w-12 h-12 text-muted-foreground" />
                  </div>
                </div>
                <h3 className="text-xl font-bold mb-2">No shared files yet</h3>
                <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
                  Files that others share with you will appear here.
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className={cn(
              viewMode === 'grid' 
                ? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6'
                : 'space-y-3'
            )}>
              {filteredFiles.map((file) => (
                <PremiumFileCard
                  key={file.file_id}
                  file={file}
                  viewMode={viewMode}
                  onDownload={handleDownload}
                  onShare={handleShare}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}

