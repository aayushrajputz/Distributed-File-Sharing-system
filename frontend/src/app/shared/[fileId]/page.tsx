'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Download, File, Clock, User, AlertCircle, CheckCircle, Eye, Calendar } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { useToast } from '@/components/ui/use-toast'

interface SharedFileData {
  file_id: string
  name: string
  size: number
  mime_type: string
  owner_name: string
  created_at: string
  expiry_time?: string
  permission: string
  is_expired: boolean
  is_valid: boolean
}

export default function SharedFilePage() {
  const params = useParams()
  const router = useRouter()
  const { toast } = useToast()
  const fileId = params.fileId as string
  
  const [fileData, setFileData] = useState<SharedFileData | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState(false)
  const [downloadProgress, setDownloadProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (fileId) {
      loadSharedFile()
    }
  }, [fileId])

  const loadSharedFile = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/${fileId}/public`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to load shared file')
      }

      const data = await response.json()
      setFileData(data)
    } catch (err: any) {
      console.error('Failed to load shared file:', err)
      setError(err.message || 'Failed to load shared file')
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async () => {
    if (!fileData) return

    try {
      setDownloading(true)
      setDownloadProgress(0)

      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/${fileId}/public/download`, {
        method: 'GET',
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Download failed')
      }

      // Get file size for progress tracking
      const contentLength = response.headers.get('Content-Length')
      const total = contentLength ? parseInt(contentLength, 10) : 0

      if (!response.body) {
        throw new Error('Response body is null')
      }

      const reader = response.body.getReader()
      const chunks: Uint8Array[] = []
      let loaded = 0

      while (true) {
        const { done, value } = await reader.read()
        
        if (done) break
        
        chunks.push(value)
        loaded += value.length
        
        if (total > 0) {
          const percentage = Math.round((loaded / total) * 100)
          setDownloadProgress(percentage)
        }
      }

      // Create blob and download
      const blob = new Blob(chunks as BlobPart[], { type: fileData.mime_type })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = fileData.name
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)

      toast({
        title: 'Download Complete',
        description: `${fileData.name} has been downloaded successfully.`,
      })
    } catch (err: any) {
      console.error('Download failed:', err)
      toast({
        title: 'Download Failed',
        description: err.message || 'Failed to download file. Please try again.',
        variant: 'destructive',
      })
    } finally {
      setDownloading(false)
      setDownloadProgress(0)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const getPermissionLabel = (permission: string) => {
    switch (permission) {
      case 'READ': return 'View Only'
      case 'WRITE': return 'View and Edit'
      case 'ADMIN': return 'Full Access'
      default: return permission
    }
  }

  const getPermissionIcon = (permission: string) => {
    switch (permission) {
      case 'READ': return <Eye className="w-4 h-4" />
      case 'WRITE': return <File className="w-4 h-4" />
      case 'ADMIN': return <User className="w-4 h-4" />
      default: return <File className="w-4 h-4" />
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardContent className="p-6">
            <div className="flex items-center justify-center space-x-2">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
              <span className="text-gray-600">Loading shared file...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error || !fileData) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-red-50 to-pink-100 flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardHeader>
            <div className="flex items-center space-x-2 text-red-600">
              <AlertCircle className="w-5 h-5" />
              <CardTitle>File Not Available</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-gray-600 mb-4">
              {error || 'The shared file could not be found or is no longer available.'}
            </p>
            <Button 
              onClick={() => router.push('/')} 
              variant="outline" 
              className="w-full"
            >
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!fileData.is_valid || fileData.is_expired) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-orange-50 to-yellow-100 flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardHeader>
            <div className="flex items-center space-x-2 text-orange-600">
              <Clock className="w-5 h-5" />
              <CardTitle>Link Expired</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-gray-600 mb-4">
              This shared file link has expired or is no longer valid.
            </p>
            <Button 
              onClick={() => router.push('/')} 
              variant="outline" 
              className="w-full"
            >
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <Card className="w-full max-w-2xl">
        <CardHeader>
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-blue-100 rounded-lg">
              <File className="w-6 h-6 text-blue-600" />
            </div>
            <div>
              <CardTitle className="text-xl">{fileData.name}</CardTitle>
              <CardDescription>
                Shared by {fileData.owner_name}
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        
        <CardContent className="space-y-6">
          {/* File Info */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex items-center space-x-2 text-sm text-gray-600">
                <File className="w-4 h-4" />
                <span>Size: {formatFileSize(fileData.size)}</span>
              </div>
              <div className="flex items-center space-x-2 text-sm text-gray-600">
                <User className="w-4 h-4" />
                <span>Owner: {fileData.owner_name}</span>
              </div>
              <div className="flex items-center space-x-2 text-sm text-gray-600">
                <Calendar className="w-4 h-4" />
                <span>Created: {formatDate(fileData.created_at)}</span>
              </div>
            </div>
            
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                {getPermissionIcon(fileData.permission)}
                <Badge variant="secondary">
                  {getPermissionLabel(fileData.permission)}
                </Badge>
              </div>
              
              {fileData.expiry_time && (
                <div className="flex items-center space-x-2 text-sm text-gray-600">
                  <Clock className="w-4 h-4" />
                  <span>Expires: {formatDate(fileData.expiry_time)}</span>
                </div>
              )}
            </div>
          </div>

          {/* Download Section */}
          <div className="border-t pt-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">Download File</h3>
              <div className="flex items-center space-x-2 text-green-600">
                <CheckCircle className="w-4 h-4" />
                <span className="text-sm">Link is valid</span>
              </div>
            </div>
            
            {downloading && (
              <div className="mb-4">
                <div className="flex items-center justify-between text-sm text-gray-600 mb-2">
                  <span>Downloading...</span>
                  <span>{downloadProgress}%</span>
                </div>
                <Progress value={downloadProgress} className="w-full" />
              </div>
            )}
            
            <Button 
              onClick={handleDownload}
              disabled={downloading || fileData.permission === 'READ' && !fileData.is_valid}
              className="w-full"
              size="lg"
            >
              <Download className="w-4 h-4 mr-2" />
              {downloading ? 'Downloading...' : 'Download File'}
            </Button>
            
            {fileData.permission === 'READ' && (
              <p className="text-sm text-gray-500 mt-2 text-center">
                You have view-only access to this file
              </p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
