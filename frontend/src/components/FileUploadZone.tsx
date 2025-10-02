'use client'

import { useCallback, useState } from 'react'
import { Upload, X, FileText, Image, Video, Music, Archive } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FileUploadZoneProps {
  onUpload: (files: FileList) => void
  disabled?: boolean
  className?: string
  maxFiles?: number
  acceptedTypes?: string[]
}

export function FileUploadZone({ 
  onUpload, 
  disabled = false, 
  className,
  maxFiles = 10,
  acceptedTypes = ['*']
}: FileUploadZoneProps) {
  const [isDragOver, setIsDragOver] = useState(false)
  const [dragReject, setDragReject] = useState(false)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])

  const getFileIcon = (file: File) => {
    if (file.type.startsWith('image/')) return <Image className="w-4 h-4 text-blue-500" />
    if (file.type.startsWith('video/')) return <Video className="w-4 h-4 text-purple-500" />
    if (file.type.startsWith('audio/')) return <Music className="w-4 h-4 text-green-500" />
    if (file.type.includes('zip') || file.type.includes('rar')) return <Archive className="w-4 h-4 text-orange-500" />
    return <FileText className="w-4 h-4 text-gray-500" />
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const validateFile = (file: File) => {
    if (acceptedTypes.includes('*')) return true
    return acceptedTypes.some(type => file.type.startsWith(type))
  }

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return
    
    setIsDragOver(true)
    setDragReject(false)
  }, [disabled])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)
    setDragReject(false)
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    
    if (disabled) return
    
    setIsDragOver(false)
    setDragReject(false)
    
    const files = Array.from(e.dataTransfer.files)
    const validFiles = files.filter(validateFile)
    
    if (validFiles.length !== files.length) {
      setDragReject(true)
      return
    }
    
    if (validFiles.length > maxFiles) {
      setDragReject(true)
      return
    }
    
    setSelectedFiles(validFiles)
    onUpload(e.dataTransfer.files)
  }, [disabled, maxFiles, onUpload])

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (disabled || !e.target.files) return
    
    const files = Array.from(e.target.files)
    const validFiles = files.filter(validateFile)
    
    if (validFiles.length !== files.length) {
      setDragReject(true)
      return
    }
    
    if (validFiles.length > maxFiles) {
      setDragReject(true)
      return
    }
    
    setSelectedFiles(validFiles)
    onUpload(e.target.files)
  }

  const removeFile = (index: number) => {
    setSelectedFiles(prev => prev.filter((_, i) => i !== index))
  }

  return (
    <div className={cn('space-y-4', className)}>
      <div
        className={cn(
          'border-2 border-dashed rounded-lg p-8 text-center transition-all duration-200 cursor-pointer',
          isDragOver && !dragReject && 'drag-over',
          dragReject && 'drag-reject',
          disabled && 'opacity-50 cursor-not-allowed',
          'hover:border-primary/50 hover:bg-primary/5'
        )}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => !disabled && document.getElementById('file-upload')?.click()}
      >
        <input
          id="file-upload"
          type="file"
          multiple
          accept={acceptedTypes.join(',')}
          onChange={handleFileSelect}
          className="hidden"
          disabled={disabled}
        />
        
        <div className="space-y-4">
          <div className="mx-auto w-12 h-12 bg-primary/10 rounded-full flex items-center justify-center">
            <Upload className="w-6 h-6 text-primary" />
          </div>
          
          <div>
            <h3 className="text-lg font-semibold">
              {dragReject ? 'Invalid files' : 'Drop files here'}
            </h3>
            <p className="text-muted-foreground">
              {dragReject 
                ? 'Some files are not supported or exceed the limit'
                : 'or click to browse your computer'
              }
            </p>
            {maxFiles > 1 && (
              <p className="text-sm text-muted-foreground mt-1">
                Up to {maxFiles} files
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Selected Files Preview */}
      {selectedFiles.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Selected Files ({selectedFiles.length})</h4>
          <div className="space-y-2 max-h-32 overflow-y-auto">
            {selectedFiles.map((file, index) => (
              <div
                key={index}
                className="flex items-center space-x-3 p-2 bg-muted rounded-lg"
              >
                {getFileIcon(file)}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{file.name}</p>
                  <p className="text-xs text-muted-foreground">{formatFileSize(file.size)}</p>
                </div>
                <button
                  onClick={() => removeFile(index)}
                  className="text-muted-foreground hover:text-destructive transition-colors"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
