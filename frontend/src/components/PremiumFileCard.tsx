'use client'

import { useState, useEffect } from 'react'
import { 
  MoreVertical, 
  Download, 
  Share2, 
  Trash2, 
  Star,
  Copy,
  Edit,
  Eye,
  FileText,
  Image as ImageIcon,
  Music,
  Video,
  Archive,
  Code,
  File as FileIcon,
  Calendar,
  HardDrive
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { FileMetadata } from '@/lib/api/files'
import { formatFileSize, formatDate } from '@/lib/utils'

interface PremiumFileCardProps {
  file: FileMetadata
  viewMode: 'grid' | 'list'
  onDownload: (fileId: string, fileName: string) => void
  onShare: (fileId: string) => void
  onDelete: (fileId: string, fileName: string) => void
  onRename?: (fileId: string, fileName: string) => void
  onFavorite?: (fileId: string) => void
  isFavorited?: boolean
}

export function PremiumFileCard({
  file,
  viewMode,
  onDownload,
  onShare,
  onDelete,
  onRename,
  onFavorite,
  isFavorited = false,
}: PremiumFileCardProps) {
  const [showMenu, setShowMenu] = useState(false)
  const [isFavorite, setIsFavorite] = useState(isFavorited)

  // Sync local state with prop changes
  useEffect(() => {
    setIsFavorite(isFavorited)
  }, [isFavorited])

  const getFileIcon = (mimeType: string) => {
    const iconClass = "w-full h-full"
    if (mimeType.startsWith('image/')) return <ImageIcon className={iconClass} />
    if (mimeType.startsWith('video/')) return <Video className={iconClass} />
    if (mimeType.startsWith('audio/')) return <Music className={iconClass} />
    if (mimeType.includes('pdf')) return <FileText className={iconClass} />
    if (mimeType.includes('zip') || mimeType.includes('rar')) return <Archive className={iconClass} />
    if (mimeType.includes('javascript') || mimeType.includes('json')) return <Code className={iconClass} />
    return <FileIcon className={iconClass} />
  }

  const getFileColor = (mimeType: string) => {
    if (mimeType.startsWith('image/')) return 'from-blue-500 to-cyan-500'
    if (mimeType.startsWith('video/')) return 'from-purple-500 to-pink-500'
    if (mimeType.startsWith('audio/')) return 'from-green-500 to-emerald-500'
    if (mimeType.includes('pdf')) return 'from-red-500 to-orange-500'
    if (mimeType.includes('zip') || mimeType.includes('rar')) return 'from-yellow-500 to-amber-500'
    if (mimeType.includes('javascript') || mimeType.includes('json')) return 'from-indigo-500 to-blue-500'
    return 'from-gray-500 to-slate-500'
  }

  const handleFavorite = () => {
    const newFavoriteState = !isFavorite
    setIsFavorite(newFavoriteState)
    onFavorite?.(file.file_id)
  }

  if (viewMode === 'list') {
    return (
      <div className="group rounded-xl border border-border/40 bg-background/50 backdrop-blur-sm hover:bg-background hover:shadow-lg hover:border-blue-600/30 transition-all duration-300">
        <div className="flex items-center p-4 space-x-4">
          {/* File Icon */}
          <div className={cn(
            'flex-shrink-0 w-12 h-12 rounded-xl bg-gradient-to-br p-2.5 text-white shadow-lg',
            getFileColor(file.mime_type)
          )}>
            {getFileIcon(file.mime_type)}
          </div>

          {/* File Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center space-x-2">
              <h3 className="text-sm font-semibold truncate group-hover:text-blue-600 transition-colors">
                {file.name}
              </h3>
              {isFavorite && <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />}
            </div>
            <div className="flex items-center space-x-3 mt-1 text-xs text-muted-foreground">
              <span className="flex items-center space-x-1">
                <HardDrive className="w-3 h-3" />
                <span>{formatFileSize(file.size)}</span>
              </span>
              <span className="flex items-center space-x-1">
                <Calendar className="w-3 h-3" />
                <span>{formatDate(file.created_at)}</span>
              </span>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onDownload(file.file_id, file.name)}
              className="h-9 w-9 rounded-lg hover:bg-blue-600/10 hover:text-blue-600"
              title="Download"
            >
              <Download className="w-4 h-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onShare(file.file_id)}
              className="h-9 w-9 rounded-lg hover:bg-green-600/10 hover:text-green-600"
              title="Share"
            >
              <Share2 className="w-4 h-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={handleFavorite}
              className="h-9 w-9 rounded-lg hover:bg-yellow-600/10 hover:text-yellow-600"
              title="Favorite"
            >
              <Star className={cn("w-4 h-4", isFavorite && "fill-yellow-500 text-yellow-500")} />
            </Button>
            
            {/* More Menu */}
            <div className="relative">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowMenu(!showMenu)}
                className="h-9 w-9 rounded-lg hover:bg-muted"
              >
                <MoreVertical className="w-4 h-4" />
              </Button>
              
              {showMenu && (
                <div className="absolute right-0 mt-2 w-48 rounded-xl border border-border/40 bg-background/95 backdrop-blur-xl shadow-2xl overflow-hidden z-50 animate-in fade-in slide-in-from-top-2 duration-200">
                  <div className="p-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        onRename?.(file.file_id, file.name)
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-muted"
                    >
                      <Edit className="w-4 h-4 mr-2" />
                      Rename
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        // Copy link functionality
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-muted"
                    >
                      <Copy className="w-4 h-4 mr-2" />
                      Copy Link
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        onDelete(file.file_id, file.name)
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-red-500/10 hover:text-red-500"
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Grid View
  return (
    <div className="group rounded-xl border border-border/40 bg-background/50 backdrop-blur-sm hover:bg-background hover:shadow-xl hover:border-blue-600/30 hover:-translate-y-1 transition-all duration-300">
      <div className="p-5 space-y-4">
        {/* Header */}
        <div className="flex items-start justify-between">
          {/* File Icon */}
          <div className={cn(
            'w-14 h-14 rounded-xl bg-gradient-to-br p-3 text-white shadow-lg transform group-hover:scale-110 transition-transform',
            getFileColor(file.mime_type)
          )}>
            {getFileIcon(file.mime_type)}
          </div>

          {/* Actions */}
          <div className="flex items-center space-x-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={handleFavorite}
              className={cn(
                "h-8 w-8 rounded-lg transition-all hover:bg-yellow-600/10",
                isFavorite ? "opacity-100" : "opacity-0 group-hover:opacity-100"
              )}
            >
              <Star className={cn("w-4 h-4", isFavorite ? "fill-yellow-500 text-yellow-500" : "text-muted-foreground")} />
            </Button>
            
            <div className="relative">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowMenu(!showMenu)}
                className="h-8 w-8 rounded-lg opacity-0 group-hover:opacity-100 transition-all"
              >
                <MoreVertical className="w-4 h-4" />
              </Button>
              
              {showMenu && (
                <div className="absolute right-0 mt-2 w-48 rounded-xl border border-border/40 bg-background/95 backdrop-blur-xl shadow-2xl overflow-hidden z-50 animate-in fade-in slide-in-from-top-2 duration-200">
                  <div className="p-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        onRename?.(file.file_id, file.name)
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-muted"
                    >
                      <Edit className="w-4 h-4 mr-2" />
                      Rename
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-muted"
                    >
                      <Copy className="w-4 h-4 mr-2" />
                      Copy Link
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        onDelete(file.file_id, file.name)
                        setShowMenu(false)
                      }}
                      className="w-full justify-start h-9 rounded-lg hover:bg-red-500/10 hover:text-red-500"
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* File Name */}
        <div>
          <h3 className="text-sm font-semibold truncate group-hover:text-blue-600 transition-colors">
            {file.name}
          </h3>
          <p className="text-xs text-muted-foreground mt-1">
            {formatFileSize(file.size)}
          </p>
        </div>

        {/* Meta Info */}
        <div className="flex items-center text-xs text-muted-foreground">
          <Calendar className="w-3 h-3 mr-1" />
          <span>{formatDate(file.created_at)}</span>
        </div>

        {/* Action Buttons */}
        <div className="flex space-x-2 pt-2 border-t border-border/40">
          <Button
            size="sm"
            variant="outline"
            onClick={() => onDownload(file.file_id, file.name)}
            className="flex-1 h-9 rounded-lg hover:bg-blue-600/10 hover:text-blue-600 hover:border-blue-600/50 transition-all"
          >
            <Download className="w-3.5 h-3.5 mr-1.5" />
            Download
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => onShare(file.file_id)}
            className="h-9 w-9 rounded-lg hover:bg-green-600/10 hover:text-green-600 hover:border-green-600/50 transition-all"
          >
            <Share2 className="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}

