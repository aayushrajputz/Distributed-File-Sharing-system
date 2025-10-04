'use client'

import { useState } from 'react'
import { AlertTriangle, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface DeleteConfirmationModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  fileName: string
  isDeleting?: boolean
}

export function DeleteConfirmationModal({
  isOpen,
  onClose,
  onConfirm,
  fileName,
  isDeleting = false
}: DeleteConfirmationModalProps) {
  const [confirmText, setConfirmText] = useState('')
  const expectedText = 'DELETE'
  const isConfirmValid = confirmText === expectedText

  const handleConfirm = () => {
    if (isConfirmValid && !isDeleting) {
      onConfirm()
    }
  }

  const handleClose = () => {
    if (!isDeleting) {
      setConfirmText('')
      onClose()
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={handleClose}
      />
      
      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-background border border-border/40 rounded-2xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border/40">
          <div className="flex items-center space-x-3">
            <div className="relative">
              <div className="absolute inset-0 bg-red-500/20 rounded-full blur-md"></div>
              <div className="relative w-10 h-10 bg-red-500/10 rounded-full flex items-center justify-center border border-red-500/20">
                <AlertTriangle className="w-5 h-5 text-red-500" />
              </div>
            </div>
            <div>
              <h2 className="text-lg font-semibold text-foreground">Delete File</h2>
              <p className="text-sm text-muted-foreground">This action cannot be undone</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleClose}
            disabled={isDeleting}
            className="h-8 w-8 hover:bg-muted/50"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          <div className="space-y-2">
            <p className="text-sm text-foreground">
              You are about to permanently delete:
            </p>
            <div className="p-3 bg-muted/50 rounded-lg border border-border/40">
              <div className="flex items-center space-x-2">
                <Trash2 className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground truncate">
                  {fileName}
                </span>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="p-4 bg-red-500/5 border border-red-500/20 rounded-lg">
              <div className="flex items-start space-x-3">
                <AlertTriangle className="w-5 h-5 text-red-500 mt-0.5 flex-shrink-0" />
                <div className="space-y-1">
                  <p className="text-sm font-medium text-red-700 dark:text-red-400">
                    Warning: This action is permanent
                  </p>
                  <p className="text-xs text-red-600 dark:text-red-500">
                    The file will be permanently deleted and cannot be recovered. There is no trash or recycle bin.
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">
                Type <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-red-600 dark:text-red-400">DELETE</span> to confirm:
              </label>
              <input
                type="text"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value.toUpperCase())}
                placeholder="Type DELETE to confirm"
                disabled={isDeleting}
                className={cn(
                  "w-full px-3 py-2 text-sm border rounded-lg bg-background transition-all",
                  "focus:outline-none focus:ring-2 focus:ring-red-500/20 focus:border-red-500/50",
                  isConfirmValid 
                    ? "border-red-500/50 text-red-600 dark:text-red-400" 
                    : "border-border/40 text-foreground",
                  isDeleting && "opacity-50 cursor-not-allowed"
                )}
                autoComplete="off"
              />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end space-x-3 p-6 border-t border-border/40">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isDeleting}
            className="hover:bg-muted/50"
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={!isConfirmValid || isDeleting}
            className={cn(
              "min-w-[100px] transition-all",
              isDeleting && "opacity-50 cursor-not-allowed"
            )}
          >
            {isDeleting ? (
              <div className="flex items-center space-x-2">
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                <span>Deleting...</span>
              </div>
            ) : (
              <div className="flex items-center space-x-2">
                <Trash2 className="w-4 h-4" />
                <span>Delete Forever</span>
              </div>
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}



