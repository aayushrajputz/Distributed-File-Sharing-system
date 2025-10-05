'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Lock, Download, Share2, Users, Grid, List, Settings } from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Sidebar } from '@/components/Sidebar'
import { PremiumHeader } from '@/components/PremiumHeader'
import { PremiumFileCard } from '@/components/PremiumFileCard'
import { FileMetadata, fileService } from '@/lib/api/files'
import { cn } from '@/lib/utils'
import { useToast } from '@/components/ui/use-toast'
import AccessManagementModal from '@/components/AccessManagementModal'

export default function PrivateFilesPage() {
  const router = useRouter()
  const { user, isAuthenticated } = useAuthStore()
  const { toast } = useToast()
  const [mounted, setMounted] = useState(false)
  const [loading, setLoading] = useState(true)
  const [privateFiles, setPrivateFiles] = useState<FileMetadata[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [downloadingFiles, setDownloadingFiles] = useState<{ [fileId: string]: { percentage: number; loaded: number; total: number; speed: number; estimatedTime: number } }>({})
  const [selectedFile, setSelectedFile] = useState<FileMetadata | null>(null)
  const [isAccessModalOpen, setIsAccessModalOpen] = useState(false)

  useEffect(() => {
    setMounted(true)
    if (!isAuthenticated()) {
      router.push('/auth/login')
      return
    }
    loadPrivateFiles()
  }, [isAuthenticated, router])

  const loadPrivateFiles = async () => {
    try {
      setLoading(true)
      const response = await fileService.listPrivateFiles(1, 100)
      setPrivateFiles(response.files)
    } catch (error) {
      console.error('Failed to load private files:', error)
      toast({
        title: 'Error',
        description: 'Failed to load private files. Please try again.',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async (fileId: string, fileName: string) => {
    try {
      setDownloadingFiles(prev => ({
        ...prev,
        [fileId]: { percentage: 0, loaded: 0, total: 0, speed: 0, estimatedTime: 0 }
      }))

      await fileService.downloadFile(fileId, fileName, (progress) => {
        setDownloadingFiles(prev => ({
          ...prev,
          [fileId]: progress
        }))
      })

      // Remove from downloading state after completion
      setDownloadingFiles(prev => {
        const newState = { ...prev }
        delete newState[fileId]
        return newState
      })

      toast({
        title: 'Download Complete',
        description: `${fileName} has been downloaded successfully.`,
      })
    } catch (error: any) {
      console.error('Download failed:', error)
      
      // Remove from downloading state
      setDownloadingFiles(prev => {
        const newState = { ...prev }
        delete newState[fileId]
        return newState
      })

      toast({
        title: 'Download Failed',
        description: error.message || 'Failed to download file. Please try again.',
        variant: 'destructive',
      })
    }
  }

  const handleShare = (fileId: string) => {
    // TODO: Implement share modal
    console.log('Share:', fileId)
  }

  const handleDelete = async (fileId: string, fileName: string) => {
    // TODO: Implement delete
    console.log('Delete:', fileId, fileName)
  }

  const handleManageAccess = (fileId: string) => {
    const file = privateFiles.find(f => f.id === fileId)
    if (file) {
      setSelectedFile(file)
      setIsAccessModalOpen(true)
    }
  }

  const handleAccessUpdated = () => {
    loadPrivateFiles()
  }

  const filteredFiles = privateFiles.filter(file =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (!mounted || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20 flex items-center justify-center">
        <div className="text-center">
          <div className="relative w-16 h-16 mx-auto mb-6">
            <div className="absolute inset-0 bg-gradient-to-r from-purple-600 to-pink-600 rounded-full blur-xl opacity-50 animate-pulse"></div>
            <div className="relative w-16 h-16 bg-gradient-to-r from-purple-600 to-pink-600 rounded-full flex items-center justify-center animate-spin">
              <div className="w-12 h-12 bg-background rounded-full"></div>
            </div>
          </div>
          <p className="text-lg font-semibold mb-2">Loading private files...</p>
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
              <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-purple-600 to-pink-600 flex items-center justify-center">
                <Lock className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-3xl font-bold bg-gradient-to-r from-purple-600 to-pink-600 bg-clip-text text-transparent">
                  Private Files
                </h1>
                <p className="text-muted-foreground">Files marked as private with restricted access</p>
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
                    viewMode === 'grid' && 'bg-gradient-to-r from-purple-600 to-pink-600 text-white'
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
                    viewMode === 'list' && 'bg-gradient-to-r from-purple-600 to-pink-600 text-white'
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
                  <div className="absolute inset-0 bg-gradient-to-r from-purple-600/20 to-pink-600/20 rounded-full blur-2xl"></div>
                  <div className="relative w-24 h-24 bg-gradient-to-r from-purple-600/10 to-pink-600/10 rounded-full flex items-center justify-center border-2 border-dashed border-purple-600/30">
                    <Lock className="w-12 h-12 text-muted-foreground" />
                  </div>
                </div>
                <h3 className="text-xl font-bold mb-2">No private files yet</h3>
                <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
                  Mark files as private to restrict access to specific users.
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
                <div key={file.file_id} className="relative">
                  <PremiumFileCard
                    file={file}
                    viewMode={viewMode}
                    onDownload={handleDownload}
                    onShare={handleShare}
                    onDelete={handleDelete}
                  />
                  {/* Privacy Badge */}
                  <div className="absolute top-2 right-2 z-10">
                    <Badge className="bg-gradient-to-r from-purple-600 to-pink-600 text-white border-0">
                      <Lock className="w-3 h-3 mr-1" />
                      Private
                    </Badge>
                  </div>
                  {/* Manage Access Button for Owners */}
                  {file.owner_id === user?.userId && (
                    <div className="absolute bottom-2 right-2 z-10">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => handleManageAccess(file.file_id)}
                        className="h-8 px-3"
                      >
                        <Settings className="w-3 h-3 mr-1" />
                        Manage Access
                      </Button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </main>

      {/* Access Management Modal */}
      {selectedFile && (
        <AccessManagementModal
          isOpen={isAccessModalOpen}
          onClose={() => {
            setIsAccessModalOpen(false)
            setSelectedFile(null)
          }}
          file={{
            id: selectedFile.id,
            name: selectedFile.name,
            shared_with: selectedFile.shared_with || [],
          }}
          onAccessUpdated={handleAccessUpdated}
        />
      )}
    </div>
  )
}

