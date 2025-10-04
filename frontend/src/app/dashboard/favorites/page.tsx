'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Star, Download, Share2, Trash2, Grid, List } from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { useNotificationStore } from '@/store/notifications'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Sidebar } from '@/components/Sidebar'
import { PremiumHeader } from '@/components/PremiumHeader'
import { PremiumFileCard } from '@/components/PremiumFileCard'
import { DeleteConfirmationModal } from '@/components/DeleteConfirmationModal'
import { FileMetadata, fileService } from '@/lib/api/files'
import { cn } from '@/lib/utils'

export default function FavoritesPage() {
  const router = useRouter()
  const { user, isAuthenticated } = useAuthStore()
  const { addNotification } = useNotificationStore()
  const [mounted, setMounted] = useState(false)
  const [loading, setLoading] = useState(true)
  const [favoriteFiles, setFavoriteFiles] = useState<FileMetadata[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [deleteFileId, setDeleteFileId] = useState<string | null>(null)
  const [deleteFileName, setDeleteFileName] = useState<string>('')
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    setMounted(true)
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadFavorites()
  }, [isAuthenticated, router])

  const loadFavorites = async () => {
    try {
      setLoading(true)
      const response = await fileService.listFavorites(1, 100)
      setFavoriteFiles(response.files)
    } catch (error) {
      console.error('Failed to load favorites:', error)
      setFavoriteFiles([])
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async (fileId: string, fileName: string) => {
    if (!user) return
    try {
      await fileService.downloadFile(fileId, fileName)
      
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'download',
        title: 'File Downloaded',
        body: `Downloaded "${fileName}"`,
        is_read: false,
        created_at: new Date().toISOString(),
      })
    } catch (error) {
      console.error('Download failed:', error)
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'error',
        title: 'Download Failed',
        body: `Failed to download "${fileName}"`,
        is_read: false,
        created_at: new Date().toISOString(),
      })
    }
  }

  const handleShare = (fileId: string) => {
    // TODO: Implement share functionality
    console.log('Share:', fileId)
  }

  const handleDelete = (fileId: string, fileName: string) => {
    setDeleteFileId(fileId)
    setDeleteFileName(fileName)
  }

  const handleConfirmDelete = async () => {
    if (!user || !deleteFileId) return
    
    setIsDeleting(true)
    try {
      await fileService.deleteFile(deleteFileId)
      await loadFavorites()
      
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'delete',
        title: 'File Deleted',
        body: `Permanently deleted "${deleteFileName}"`,
        is_read: false,
        created_at: new Date().toISOString(),
      })
      
      // Close modal
      setDeleteFileId(null)
      setDeleteFileName('')
    } catch (error) {
      console.error('Delete failed:', error)
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'error',
        title: 'Delete Failed',
        body: `Failed to delete "${deleteFileName}"`,
        is_read: false,
        created_at: new Date().toISOString(),
      })
    } finally {
      setIsDeleting(false)
    }
  }

  const handleCancelDelete = () => {
    if (!isDeleting) {
      setDeleteFileId(null)
      setDeleteFileName('')
    }
  }

  const handleFavorite = async (fileId: string) => {
    if (!user) return
    try {
      // Since this is the favorites page, clicking the star should remove from favorites
      await fileService.removeFromFavorites(fileId)
      await loadFavorites()
      
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'info',
        title: 'Removed from Favorites',
        body: 'File removed from favorites',
        is_read: false,
        created_at: new Date().toISOString(),
      })
    } catch (error) {
      console.error('Failed to remove from favorites:', error)
    }
  }

  const filteredFiles = favoriteFiles.filter(file =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (!mounted || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20 flex items-center justify-center">
        <div className="text-center">
          <div className="relative w-16 h-16 mx-auto mb-6">
            <div className="absolute inset-0 bg-gradient-to-r from-yellow-600 to-amber-600 rounded-full blur-xl opacity-50 animate-pulse"></div>
            <div className="relative w-16 h-16 bg-gradient-to-r from-yellow-600 to-amber-600 rounded-full flex items-center justify-center animate-spin">
              <div className="w-12 h-12 bg-background rounded-full"></div>
            </div>
          </div>
          <p className="text-lg font-semibold mb-2">Loading favorites...</p>
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
              <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-yellow-600 to-amber-600 flex items-center justify-center">
                <Star className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-3xl font-bold bg-gradient-to-r from-yellow-600 to-amber-600 bg-clip-text text-transparent">
                  Favorites
                </h1>
                <p className="text-muted-foreground">Your starred files for quick access</p>
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
                    viewMode === 'grid' && 'bg-gradient-to-r from-yellow-600 to-amber-600 text-white'
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
                    viewMode === 'list' && 'bg-gradient-to-r from-yellow-600 to-amber-600 text-white'
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
                  <div className="absolute inset-0 bg-gradient-to-r from-yellow-600/20 to-amber-600/20 rounded-full blur-2xl"></div>
                  <div className="relative w-24 h-24 bg-gradient-to-r from-yellow-600/10 to-amber-600/10 rounded-full flex items-center justify-center border-2 border-dashed border-yellow-600/30">
                    <Star className="w-12 h-12 text-muted-foreground" />
                  </div>
                </div>
                <h3 className="text-xl font-bold mb-2">No favorites yet</h3>
                <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
                  Star your important files to find them quickly here.
                </p>
                <Button 
                  onClick={() => router.push('/dashboard')}
                  className="bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-700 hover:to-amber-700 text-white"
                >
                  Browse Files
                </Button>
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
                  onFavorite={handleFavorite}
                  isFavorited={true}
                />
              ))}
            </div>
          )}
        </div>
      </main>

      {/* Delete Confirmation Modal */}
      <DeleteConfirmationModal
        isOpen={!!deleteFileId}
        onClose={handleCancelDelete}
        onConfirm={handleConfirmDelete}
        fileName={deleteFileName}
        isDeleting={isDeleting}
      />
    </div>
  )
}

