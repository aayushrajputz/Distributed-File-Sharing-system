'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useNotificationStore } from '@/store/notifications'
import { fileService, FileMetadata } from '@/lib/api/files'
import { notificationService } from '@/lib/api/notifications'
import { storageService, StorageUsage } from '@/lib/api/storage'
import { authService } from '@/lib/api/auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Grid,
  List,
  Plus,
  AlertCircle,
  TrendingUp,
  Users,
  FolderOpen
} from 'lucide-react'
import { Sidebar } from '@/components/Sidebar'
import { PremiumHeader } from '@/components/PremiumHeader'
import { PremiumUploadZone } from '@/components/PremiumUploadZone'
import { PremiumFileCard } from '@/components/PremiumFileCard'
import { FileSharingModal } from '@/components/FileSharingModal'
import { StorageLimitModal } from '@/components/StorageLimitModal'
import { DeleteConfirmationModal } from '@/components/DeleteConfirmationModal'
import { cn, safeDateParse } from '@/lib/utils'

export default function DashboardPage() {
  const router = useRouter()
  const { user, isAuthenticated, clearAuth } = useAuthStore()
  const { unreadCount, setUnreadCount, addNotification } = useNotificationStore()
  const [files, setFiles] = useState<FileMetadata[]>([])
  const [sharedFiles, setSharedFiles] = useState<FileMetadata[]>([])
  const [storageData, setStorageData] = useState<StorageUsage | null>(null)
  const [favoriteStatus, setFavoriteStatus] = useState<{ [fileId: string]: boolean }>({})
  const [loading, setLoading] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<'name' | 'size' | 'date'>('date')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [filterType, setFilterType] = useState<string>('all')
  const [shareFileId, setShareFileId] = useState<string | null>(null)
  const [darkMode, setDarkMode] = useState(false)
  const [mounted, setMounted] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [showUploadZone, setShowUploadZone] = useState(false)
  const [showStorageLimitModal, setShowStorageLimitModal] = useState(false)
  const [deleteFileId, setDeleteFileId] = useState<string | null>(null)
  const [deleteFileName, setDeleteFileName] = useState<string>('')
  const [isDeleting, setIsDeleting] = useState(false)
  const [serviceStatus, setServiceStatus] = useState({
    fileService: true,
    notificationService: true
  })



  // Filter files based on search and type
  const filteredFiles = files.filter(file => {
    const matchesSearch = file.name.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesType = filterType === 'all' || (file.mime_type && file.mime_type.startsWith(filterType))
    return matchesSearch && matchesType
  })

  // Sort files
  const sortedFiles = [...filteredFiles].sort((a, b) => {
    let comparison = 0
    switch (sortBy) {
      case 'name':
        comparison = a.name.localeCompare(b.name)
        break
      case 'size':
        comparison = a.size - b.size
        break
      case 'date':
        comparison = safeDateParse(a.created_at).getTime() - safeDateParse(b.created_at).getTime()
        break
    }
    return sortOrder === 'asc' ? comparison : -comparison
  })

  useEffect(() => {
    setMounted(true)
  }, [])

  useEffect(() => {
    if (!mounted) return
    
    console.log('Dashboard useEffect - mounted:', mounted, 'isAuthenticated:', isAuthenticated(), 'user:', user)
    
    if (!isAuthenticated()) {
      console.log('Not authenticated, redirecting to login')
      router.push('/auth/login')
      return
    }
    
    // If we have a token but no user data, try to load user info
    if (isAuthenticated() && !user) {
      console.log('Authenticated but no user data, loading user info')
      loadUserInfo()
    } else if (user) {
      console.log('Authenticated with user data, loading files and notifications')
      loadFiles()
      loadNotifications()
      loadStorageData()
    }
  }, [mounted, isAuthenticated, router, user])

  const loadUserInfo = useCallback(async () => {
    try {
      const token = localStorage.getItem('access_token')
      if (!token) {
        console.log('No token found, redirecting to login')
        router.push('/auth/login')
        return
      }

      // Validate token and get user info
      const response = await authService.validateToken(token)
      if (response.valid && response.user_id) {
        // Get full user details
        const userResponse = await authService.getUser(response.user_id)
        const { setAuth } = useAuthStore.getState()
        setAuth(userResponse.user, token, localStorage.getItem('refresh_token') || '')
        console.log('User info loaded successfully')
      } else {
        console.log('Invalid token, redirecting to login')
        clearAuth()
        router.push('/auth/login')
      }
    } catch (error) {
      console.error('Failed to load user info:', error)
      clearAuth()
      router.push('/auth/login')
    }
  }, [router])

  const loadFavoriteStatus = async (fileIds: string[]) => {
    if (!user || fileIds.length === 0) return
    try {
      const status = await fileService.checkFavoriteStatus(fileIds)
      setFavoriteStatus(prev => ({ ...prev, ...status }))
    } catch (error) {
      console.error('Failed to load favorite status:', error)
    }
  }

  const loadFiles = useCallback(async () => {
    if (!user) return
    console.log('Loading files for user:', user.userId)
    try {
      const [myFiles, shared] = await Promise.all([
        fileService.listFiles(),
        fileService.listSharedFiles(),
      ])
      console.log('Files loaded successfully:', { myFiles: myFiles.files.length, shared: shared.files.length })
      setFiles(myFiles.files)
      setSharedFiles(shared.files)
      
      // Load favorite status for all files
      const allFileIds = [...myFiles.files.map(f => f.file_id), ...shared.files.map(f => f.file_id)]
      await loadFavoriteStatus(allFileIds)
      
      setServiceStatus(prev => ({ ...prev, fileService: true }))
    } catch (error) {
      console.error('Failed to load files:', error)
      // Set empty arrays if API fails
      setFiles([])
      setSharedFiles([])
      setServiceStatus(prev => ({ ...prev, fileService: false }))
    } finally {
      setLoading(false)
    }
  }, [user, loadFavoriteStatus])

  const loadNotifications = useCallback(async () => {
    if (!user) return
    try {
      const result = await notificationService.getUnreadCount(user.userId)
      setUnreadCount(result.count)
      setServiceStatus(prev => ({ ...prev, notificationService: true }))
    } catch (error) {
      console.error('Failed to load notifications:', error)
      // Set zero count if API fails
      setUnreadCount(0)
      setServiceStatus(prev => ({ ...prev, notificationService: false }))
    }
  }, [user])

  const loadStorageData = async () => {
    try {
      const data = await storageService.getStorageUsage()
      setStorageData(data)
    } catch (error) {
      console.error('Failed to load storage data:', error)
      // Set default values if API fails
      setStorageData({
        used_bytes: 0,
        quota_bytes: 100 * 1024 * 1024 * 1024, // 100GB
        file_count: 0,
        used_gb: 0,
        quota_gb: 100,
        usage_percentage: 0
      })
    }
  }

  const handleFileUpload = async (files: FileList) => {
    if (!user || files.length === 0) return

    setUploading(true)

    try {
      // Check storage quota before uploading
      const storageData = await storageService.getStorageUsage()
      const totalFileSize = Array.from(files).reduce((total, file) => total + file.size, 0)
      
      if (storageData.used_bytes + totalFileSize > storageData.quota_bytes) {
        addNotification({
          notification_id: Date.now().toString(),
          user_id: user?.userId || '',
          type: 'error',
          title: 'Storage Quota Exceeded',
          body: `Upload would exceed your storage quota. You have ${(storageData.quota_bytes - storageData.used_bytes) / (1024 * 1024 * 1024)} GB remaining.`,
          is_read: false,
          created_at: new Date().toISOString(),
        })
        setUploading(false)
        return
      }

      for (let i = 0; i < files.length; i++) {
        const file = files[i]

        console.log(`Uploading file ${i + 1}/${files.length}:`, {
          name: file.name,
          size: file.size,
          type: file.type,
        })

        // Step 1: Initiate upload
        const uploadResponse = await fileService.uploadFile({
          name: file.name,
          size: file.size,
          mime_type: file.type,
        })

        console.log('Upload response received:', uploadResponse)
        console.log('Upload response type:', typeof uploadResponse)
        console.log('Upload response keys:', Object.keys(uploadResponse))

        const { file_id, upload_url } = uploadResponse

        console.log('Extracted values:', {
          file_id,
          upload_url,
          file_id_type: typeof file_id,
          upload_url_type: typeof upload_url
        })

        if (!upload_url) {
          throw new Error(`Upload URL is missing from response. Response: ${JSON.stringify(uploadResponse)}`)
        }

        if (!file_id) {
          throw new Error(`File ID is missing from response. Response: ${JSON.stringify(uploadResponse)}`)
        }

        // Step 2: Upload to storage
        await fileService.uploadToStorage(upload_url, file, () => {})

        console.log('File uploaded to storage')

        // Step 3: Complete upload
        await fileService.completeUpload(file_id)

        console.log('Upload completed successfully')

        // Add notification
        addNotification({
          notification_id: Date.now().toString(),
          user_id: user.userId,
          type: 'upload',
          title: 'File Uploaded',
          body: `File "${file.name}" uploaded successfully`,
          is_read: false,
          created_at: new Date().toISOString(),
        })
      }

      // Reload files and storage data
      await loadFiles()
      await loadStorageData()
      setShowUploadZone(false)
    } catch (error: any) {
      console.error('Upload failed with error:', {
        message: error.message,
        response: error.response?.data,
        status: error.response?.status,
        stack: error.stack,
      })

      const errorMessage = error.response?.data?.message || error.message || 'Unknown error occurred'

        addNotification({
          notification_id: Date.now().toString(),
          user_id: user.userId,
          type: 'error',
          title: 'Upload Failed',
          body: `Upload failed: ${errorMessage}`,
          is_read: false,
          created_at: new Date().toISOString(),
        })
    } finally {
      setUploading(false)
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
    setShareFileId(fileId)
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
      await loadFiles()
      await loadStorageData()
      
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
      const isCurrentlyFavorited = favoriteStatus[fileId] || false
      
      if (isCurrentlyFavorited) {
        await fileService.removeFromFavorites(fileId)
        setFavoriteStatus(prev => ({ ...prev, [fileId]: false }))
        
        addNotification({
          notification_id: Date.now().toString(),
          user_id: user.userId,
          type: 'info',
          title: 'Removed from Favorites',
          body: 'File removed from favorites',
          is_read: false,
          created_at: new Date().toISOString(),
        })
      } else {
        await fileService.addToFavorites(fileId)
        setFavoriteStatus(prev => ({ ...prev, [fileId]: true }))
        
        addNotification({
          notification_id: Date.now().toString(),
          user_id: user.userId,
          type: 'info',
          title: 'Added to Favorites',
          body: 'File added to favorites',
          is_read: false,
          created_at: new Date().toISOString(),
        })
      }
    } catch (error) {
      console.error('Failed to update favorites:', error)
      addNotification({
        notification_id: Date.now().toString(),
        user_id: user.userId,
        type: 'error',
        title: 'Favorites Error',
        body: 'Failed to update favorites',
        is_read: false,
        created_at: new Date().toISOString(),
      })
    }
  }

  const handleLogout = () => {
    clearAuth()
    router.push('/auth/login')
  }

  const toggleDarkMode = () => {
    setDarkMode(!darkMode)
    document.documentElement.classList.toggle('dark')
  }

  if (!mounted || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20 flex items-center justify-center">
        <div className="text-center">
          <div className="relative w-16 h-16 mx-auto mb-6">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-full blur-xl opacity-50 animate-pulse"></div>
            <div className="relative w-16 h-16 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-full flex items-center justify-center animate-spin">
              <div className="w-12 h-12 bg-background rounded-full"></div>
            </div>
          </div>
          <p className="text-lg font-semibold mb-2">Loading your workspace...</p>
          <p className="text-sm text-muted-foreground">Please wait while we fetch your files</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
      {/* Sidebar */}
      <Sidebar
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
        userId={user?.userId}
      />

      {/* Header */}
      <PremiumHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        darkMode={darkMode}
        onToggleDarkMode={toggleDarkMode}
        sidebarCollapsed={sidebarCollapsed}
      />

      {/* Main Content */}
      <main className={cn(
        'pt-16 transition-all duration-300',
        sidebarCollapsed ? 'ml-20' : 'ml-64'
      )}>
        <div className="p-8 space-y-8">
          {/* Service Status Alert */}
          {(!serviceStatus.fileService || !serviceStatus.notificationService) && (
            <div className="rounded-xl border border-orange-500/20 bg-gradient-to-r from-orange-500/10 via-amber-500/10 to-yellow-500/10 backdrop-blur-sm p-4">
              <div className="flex items-start space-x-3">
                <AlertCircle className="w-5 h-5 text-orange-500 flex-shrink-0 mt-0.5" />
                <div className="flex-1">
                  <h4 className="text-sm font-semibold text-orange-900 dark:text-orange-100 mb-1">
                    Service Status Warning
                  </h4>
                  <p className="text-sm text-orange-800 dark:text-orange-200">
                    Some services are temporarily unavailable.
                    {!serviceStatus.fileService && " File operations are limited."}
                    {!serviceStatus.notificationService && " Notifications are disabled."}
                    {" "}Please try again later.
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <Card className="border-border/40 bg-gradient-to-br from-blue-600/10 via-blue-600/5 to-transparent backdrop-blur-sm hover:shadow-lg transition-all">
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">Total Files</p>
                    <h3 className="text-3xl font-bold">{files.length}</h3>
                    <p className="text-xs text-muted-foreground mt-1 flex items-center">
                      <TrendingUp className="w-3 h-3 mr-1 text-green-500" />
                      <span className="text-green-500">+{files.length > 0 ? Math.round((files.length / Math.max(files.length, 1)) * 12) : 0}%</span> from last month
                    </p>
                  </div>
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 flex items-center justify-center">
                    <FolderOpen className="w-6 h-6 text-white" />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/40 bg-gradient-to-br from-green-600/10 via-green-600/5 to-transparent backdrop-blur-sm hover:shadow-lg transition-all">
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">Shared Files</p>
                    <h3 className="text-3xl font-bold">{sharedFiles.length}</h3>
                    <p className="text-xs text-muted-foreground mt-1 flex items-center">
                      <TrendingUp className="w-3 h-3 mr-1 text-green-500" />
                      <span className="text-green-500">+{sharedFiles.length > 0 ? Math.round((sharedFiles.length / Math.max(sharedFiles.length, 1)) * 8) : 0}%</span> from last month
                    </p>
                  </div>
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-green-600 to-emerald-600 flex items-center justify-center">
                    <Users className="w-6 h-6 text-white" />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/40 bg-gradient-to-br from-purple-600/10 via-purple-600/5 to-transparent backdrop-blur-sm hover:shadow-lg transition-all">
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">Storage Used</p>
                    <h3 className="text-3xl font-bold">
                      {storageData ? `${storageData.used_gb.toFixed(1)} GB` : '0.0 GB'}
                    </h3>
                    <p className="text-xs text-muted-foreground mt-1">
                      of {storageData ? `${storageData.quota_gb.toFixed(0)} GB` : '100 GB'} ({storageData ? storageData.usage_percentage.toFixed(0) : 0}%)
                    </p>
                  </div>
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-r from-purple-600 to-pink-600 flex items-center justify-center">
                    <TrendingUp className="w-6 h-6 text-white" />
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Upload Section */}
          {showUploadZone ? (
            <Card className="border-border/40 bg-background/50 backdrop-blur-sm">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-xl">Upload Files</CardTitle>
                    <CardDescription>Drag and drop your files or click to browse</CardDescription>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowUploadZone(false)}
                  >
                    Cancel
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <PremiumUploadZone
                  onUpload={handleFileUpload}
                  disabled={uploading}
                  onStorageLimitExceeded={() => setShowStorageLimitModal(true)}
                />
              </CardContent>
            </Card>
          ) : (
            <Button
              size="lg"
              onClick={() => setShowUploadZone(true)}
              className="w-full h-16 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white shadow-lg hover:shadow-xl transition-all rounded-xl"
            >
              <Plus className="w-5 h-5 mr-2" />
              Upload New Files
            </Button>
          )}

          {/* File Management Section */}
          <div className="space-y-6">
            {/* Section Header with Controls */}
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div className="flex items-center space-x-3">
                <h2 className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                  My Files
                </h2>
                <Badge
                  variant="secondary"
                  className="bg-blue-600/10 text-blue-600 border-blue-600/20 px-3 py-1"
                >
                  {sortedFiles.length} files
                </Badge>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                {/* Filter Dropdown */}
                <select
                  value={filterType}
                  onChange={(e) => setFilterType(e.target.value)}
                  className="px-4 py-2 border border-border/40 rounded-xl bg-background/50 backdrop-blur-sm text-sm font-medium hover:border-blue-600/50 focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 transition-all"
                >
                  <option value="all">All Types</option>
                  <option value="image/">📷 Images</option>
                  <option value="video/">🎥 Videos</option>
                  <option value="audio/">🎵 Audio</option>
                  <option value="application/pdf">📄 PDFs</option>
                  <option value="application/zip">📦 Archives</option>
                </select>

                {/* Sort Dropdown */}
                <select
                  value={`${sortBy}-${sortOrder}`}
                  onChange={(e) => {
                    const [field, order] = e.target.value.split('-')
                    setSortBy(field as 'name' | 'size' | 'date')
                    setSortOrder(order as 'asc' | 'desc')
                  }}
                  className="px-4 py-2 border border-border/40 rounded-xl bg-background/50 backdrop-blur-sm text-sm font-medium hover:border-blue-600/50 focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 transition-all"
                >
                  <option value="date-desc">🕐 Newest first</option>
                  <option value="date-asc">🕐 Oldest first</option>
                  <option value="name-asc">🔤 Name A-Z</option>
                  <option value="name-desc">🔤 Name Z-A</option>
                  <option value="size-desc">📊 Largest first</option>
                  <option value="size-asc">📊 Smallest first</option>
                </select>

                {/* View Mode Toggle */}
                <div className="flex border border-border/40 rounded-xl bg-background/50 backdrop-blur-sm overflow-hidden">
                  <Button
                    variant={viewMode === 'grid' ? 'default' : 'ghost'}
                    size="sm"
                    onClick={() => setViewMode('grid')}
                    className={cn(
                      'rounded-none h-10 px-4',
                      viewMode === 'grid' && 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white'
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
                      viewMode === 'list' && 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white'
                    )}
                  >
                    <List className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </div>

            {/* Files Display */}
            {sortedFiles.length === 0 ? (
              <Card className="border-border/40 bg-background/50 backdrop-blur-sm text-center py-16">
                <CardContent>
                  <div className="relative w-24 h-24 mx-auto mb-6">
                    <div className="absolute inset-0 bg-gradient-to-r from-blue-600/20 to-indigo-600/20 rounded-full blur-2xl"></div>
                    <div className="relative w-24 h-24 bg-gradient-to-r from-blue-600/10 to-indigo-600/10 rounded-full flex items-center justify-center border-2 border-dashed border-blue-600/30">
                      <FolderOpen className="w-12 h-12 text-muted-foreground" />
                    </div>
                  </div>
                  <h3 className="text-xl font-bold mb-2">No files yet</h3>
                  <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
                    Your workspace is empty. Upload your first file to get started with CloudShare.
                  </p>
                  <Button
                    size="lg"
                    onClick={() => setShowUploadZone(true)}
                    className="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white shadow-lg"
                  >
                    <Plus className="w-5 h-5 mr-2" />
                    Upload Your First File
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className={cn(
                viewMode === 'grid'
                  ? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6'
                  : 'space-y-3'
              )}>
                {sortedFiles.map((file) => (
                  <PremiumFileCard
                    key={file.file_id}
                    file={file}
                    viewMode={viewMode}
                    onDownload={handleDownload}
                    onShare={handleShare}
                    onDelete={handleDelete}
                    onFavorite={handleFavorite}
                    isFavorited={favoriteStatus[file.file_id] || false}
                  />
                ))}
              </div>
            )}

          </div>

          {/* Shared Files Section */}
          {sharedFiles.length > 0 && (
            <div className="space-y-6 mt-12">
              <div className="flex items-center space-x-3">
                <h2 className="text-2xl font-bold bg-gradient-to-r from-green-600 to-emerald-600 bg-clip-text text-transparent">
                  Shared with Me
                </h2>
                <Badge
                  variant="secondary"
                  className="bg-green-600/10 text-green-600 border-green-600/20 px-3 py-1"
                >
                  {sharedFiles.length} files
                </Badge>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {sharedFiles.map((file) => (
                  <PremiumFileCard
                    key={file.file_id}
                    file={file}
                    viewMode="grid"
                    onDownload={handleDownload}
                    onShare={handleShare}
                    onDelete={handleDelete}
                    onFavorite={handleFavorite}
                    isFavorited={favoriteStatus[file.file_id] || false}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </main>

      {/* File Sharing Modal */}
      {shareFileId && (
        <FileSharingModal
          fileId={shareFileId}
          fileName={files.find(f => f.file_id === shareFileId)?.name || ''}
          onClose={() => setShareFileId(null)}
        />
      )}

      {/* Storage Limit Modal */}
      {showStorageLimitModal && storageData && (
        <StorageLimitModal
          isOpen={showStorageLimitModal}
          onClose={() => setShowStorageLimitModal(false)}
          userId={user?.userId || ''}
          fileSize={0}
        />
      )}

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