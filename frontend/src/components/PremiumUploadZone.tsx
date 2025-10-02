'use client'

import { useCallback, useState } from 'react'
import { Upload, File, X, CheckCircle2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { StorageLimitModal } from '@/components/StorageLimitModal'
import { cn } from '@/lib/utils'

interface UploadFile {
  id: string
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
}

interface PremiumUploadZoneProps {
  onUpload: (files: FileList) => Promise<void>
  disabled?: boolean
  className?: string
  onStorageLimitExceeded?: () => void
}

export function PremiumUploadZone({ onUpload, disabled, className, onStorageLimitExceeded }: PremiumUploadZoneProps) {
  const [isDragging, setIsDragging] = useState(false)
  const [uploadFiles, setUploadFiles] = useState<UploadFile[]>([])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (!disabled) {
      setIsDragging(true)
    }
  }, [disabled])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback(
    async (e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setIsDragging(false)

      if (disabled) return

      const files = e.dataTransfer.files
      if (files.length > 0) {
        await handleFiles(files)
      }
    },
    [disabled]
  )

  const handleFileInput = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files
      if (files && files.length > 0) {
        await handleFiles(files)
      }
      // Reset input
      e.target.value = ''
    },
    []
  )

  const handleFiles = async (files: FileList) => {
    try {
      // Check storage quota before upload
      const totalSize = Array.from(files).reduce((sum, file) => sum + file.size, 0)
      
      // Import storage service dynamically to avoid circular imports
      const { storageService } = await import('@/lib/api/storage')
      const storageData = await storageService.getStorageUsage()
      
      if (storageData.used_bytes + totalSize > storageData.quota_bytes) {
        if (onStorageLimitExceeded) {
          onStorageLimitExceeded()
        }
        return
      }

      // Create upload file objects
      const newUploadFiles: UploadFile[] = Array.from(files).map((file) => ({
        id: Math.random().toString(36).substring(7),
        file,
        progress: 0,
        status: 'pending' as const,
      }))

      setUploadFiles((prev) => [...prev, ...newUploadFiles])

      // Call the actual upload function
      await onUpload(files)

      // Simulate upload progress (replace with actual upload logic)
      for (const uploadFile of newUploadFiles) {
        setUploadFiles((prev) =>
          prev.map((f) =>
            f.id === uploadFile.id ? { ...f, status: 'uploading' as const } : f
          )
        )

        // Simulate progress
        for (let i = 0; i <= 100; i += 10) {
          await new Promise((resolve) => setTimeout(resolve, 100))
          setUploadFiles((prev) =>
            prev.map((f) =>
              f.id === uploadFile.id ? { ...f, progress: i } : f
            )
          )
        }

        setUploadFiles((prev) =>
          prev.map((f) =>
            f.id === uploadFile.id ? { ...f, status: 'success' as const, progress: 100 } : f
          )
        )
      }

      // Clear completed uploads after 3 seconds
      setTimeout(() => {
        setUploadFiles((prev) => prev.filter((f) => f.status !== 'success'))
      }, 3000)
    } catch (error) {
      console.error('Upload failed:', error)
      // Mark files as error
      setUploadFiles((prev) =>
        prev.map((f) =>
          f.status === 'uploading' ? { ...f, status: 'error' as const, error: 'Upload failed' } : f
        )
      )
    }
  }

  const removeFile = (id: string) => {
    setUploadFiles((prev) => prev.filter((f) => f.id !== id))
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  return (
    <div className={cn('space-y-4', className)}>
      {/* Drop Zone */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={cn(
          'relative rounded-2xl border-2 border-dashed transition-all duration-300 overflow-hidden',
          isDragging
            ? 'border-blue-600 bg-blue-600/10 scale-[1.02]'
            : 'border-border/40 bg-gradient-to-br from-muted/30 via-muted/20 to-transparent hover:border-blue-600/50 hover:bg-muted/40',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        {/* Animated Background */}
        <div className="absolute inset-0 bg-gradient-to-r from-blue-600/5 via-indigo-600/5 to-purple-600/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
        
        <div className="relative p-12">
          <input
            type="file"
            id="file-upload"
            multiple
            onChange={handleFileInput}
            disabled={disabled}
            className="hidden"
          />
          
          <label
            htmlFor="file-upload"
            className={cn(
              'flex flex-col items-center justify-center cursor-pointer',
              disabled && 'cursor-not-allowed'
            )}
          >
            {/* Upload Icon with Animation */}
            <div className="relative mb-6">
              <div className="absolute inset-0 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-full blur-xl opacity-30 animate-pulse" />
              <div className="relative w-20 h-20 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-full flex items-center justify-center shadow-lg transform transition-transform hover:scale-110">
                <Upload className="w-10 h-10 text-white" />
              </div>
            </div>

            {/* Text */}
            <div className="text-center space-y-2">
              <h3 className="text-xl font-semibold">
                {isDragging ? 'Drop files here' : 'Upload your files'}
              </h3>
              <p className="text-sm text-muted-foreground max-w-sm">
                Drag and drop files here, or click to browse
              </p>
              <p className="text-xs text-muted-foreground">
                Supports: Images, Videos, Documents, Archives (Max 100MB per file)
              </p>
            </div>

            {/* Button */}
            <Button
              type="button"
              size="lg"
              className="mt-6 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white shadow-lg hover:shadow-xl transition-all"
              disabled={disabled}
            >
              <Upload className="w-4 h-4 mr-2" />
              Choose Files
            </Button>
          </label>
        </div>
      </div>

      {/* Upload Progress List */}
      {uploadFiles.length > 0 && (
        <div className="space-y-2">
          {uploadFiles.map((uploadFile) => (
            <div
              key={uploadFile.id}
              className="rounded-xl border border-border/40 bg-background/50 backdrop-blur-sm p-4 transition-all duration-300 hover:shadow-md"
            >
              <div className="flex items-center space-x-4">
                {/* File Icon */}
                <div className="flex-shrink-0">
                  {uploadFile.status === 'success' ? (
                    <CheckCircle2 className="w-8 h-8 text-green-500" />
                  ) : uploadFile.status === 'error' ? (
                    <AlertCircle className="w-8 h-8 text-red-500" />
                  ) : (
                    <File className="w-8 h-8 text-blue-600" />
                  )}
                </div>

                {/* File Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium truncate">
                      {uploadFile.file.name}
                    </p>
                    <span className="text-xs text-muted-foreground ml-2">
                      {formatFileSize(uploadFile.file.size)}
                    </span>
                  </div>

                  {/* Progress Bar */}
                  {uploadFile.status === 'uploading' && (
                    <div className="space-y-1">
                      <Progress value={uploadFile.progress} className="h-1.5" />
                      <p className="text-xs text-muted-foreground">
                        Uploading... {uploadFile.progress}%
                      </p>
                    </div>
                  )}

                  {uploadFile.status === 'success' && (
                    <p className="text-xs text-green-500 font-medium">
                      Upload complete
                    </p>
                  )}

                  {uploadFile.status === 'error' && (
                    <p className="text-xs text-red-500">
                      {uploadFile.error || 'Upload failed'}
                    </p>
                  )}
                </div>

                {/* Remove Button */}
                {uploadFile.status !== 'uploading' && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => removeFile(uploadFile.id)}
                    className="flex-shrink-0 h-8 w-8 rounded-lg hover:bg-red-500/10 hover:text-red-500"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

