'use client'

import { useState, useEffect } from 'react'
import { X, Search, UserPlus, UserMinus, Mail, Clock, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { fileService } from '@/lib/api/files'
import { useToast } from '@/components/ui/use-toast'

interface User {
  id: string
  email: string
  name?: string
}

interface PrivateFileAccessModalProps {
  fileId: string
  fileName: string
  currentSharedWith: string[]
  isOpen: boolean
  onClose: () => void
  onUpdate: () => void
}

export function PrivateFileAccessModal({
  fileId,
  fileName,
  currentSharedWith,
  isOpen,
  onClose,
  onUpdate,
}: PrivateFileAccessModalProps) {
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<User[]>([])
  const [sharedUsers, setSharedUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [searching, setSearching] = useState(false)
  const [expiryTime, setExpiryTime] = useState('')

  useEffect(() => {
    if (isOpen) {
      // Load current shared users
      loadSharedUsers()
    }
  }, [isOpen, currentSharedWith])

  const loadSharedUsers = async () => {
    // For now, just create user objects from IDs
    // In a real implementation, you'd fetch user details from auth-service
    const users = currentSharedWith.map(userId => ({
      id: userId,
      email: `user-${userId}@example.com`, // Placeholder
      name: `User ${userId.substring(0, 8)}`,
    }))
    setSharedUsers(users)
  }

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      setSearchResults([])
      return
    }

    setSearching(true)
    try {
      // TODO: Implement actual user search via auth-service
      // For now, simulate search results
      await new Promise(resolve => setTimeout(resolve, 500))
      
      const mockResults: User[] = [
        { id: 'user123', email: searchQuery, name: 'John Doe' },
        { id: 'user456', email: `${searchQuery}.test`, name: 'Jane Smith' },
      ]
      
      // Filter out already shared users
      const filtered = mockResults.filter(
        user => !currentSharedWith.includes(user.id)
      )
      
      setSearchResults(filtered)
    } catch (error) {
      console.error('Search failed:', error)
      toast({
        title: 'Search Failed',
        description: 'Failed to search for users. Please try again.',
        variant: 'destructive',
      })
    } finally {
      setSearching(false)
    }
  }

  const handleAddUser = async (userId: string) => {
    setLoading(true)
    try {
      await fileService.managePrivateAccess(fileId, [userId], 'add')
      
      toast({
        title: 'User Added',
        description: 'User has been granted access to this private file.',
      })
      
      // Update local state
      const user = searchResults.find(u => u.id === userId)
      if (user) {
        setSharedUsers([...sharedUsers, user])
      }
      
      // Remove from search results
      setSearchResults(searchResults.filter(u => u.id !== userId))
      
      onUpdate()
    } catch (error: any) {
      console.error('Failed to add user:', error)
      toast({
        title: 'Failed to Add User',
        description: error.message || 'Failed to grant access. Please try again.',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleRemoveUser = async (userId: string) => {
    setLoading(true)
    try {
      await fileService.managePrivateAccess(fileId, [userId], 'remove')
      
      toast({
        title: 'User Removed',
        description: 'User access has been revoked.',
      })
      
      // Update local state
      setSharedUsers(sharedUsers.filter(u => u.id !== userId))
      
      onUpdate()
    } catch (error: any) {
      console.error('Failed to remove user:', error)
      toast({
        title: 'Failed to Remove User',
        description: error.message || 'Failed to revoke access. Please try again.',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <Card className="w-full max-w-2xl max-h-[80vh] overflow-hidden bg-background border-border/40 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border/40 bg-gradient-to-r from-purple-600/10 to-pink-600/10">
          <div>
            <h2 className="text-2xl font-bold bg-gradient-to-r from-purple-600 to-pink-600 bg-clip-text text-transparent">
              Manage Access
            </h2>
            <p className="text-sm text-muted-foreground mt-1">
              {fileName}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="rounded-full"
          >
            <X className="w-5 h-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 overflow-y-auto max-h-[calc(80vh-140px)]">
          {/* Search Section */}
          <div className="space-y-3">
            <label className="text-sm font-semibold">Add Users</label>
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="Search by email..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                  className="pl-10"
                />
              </div>
              <Button
                onClick={handleSearch}
                disabled={searching || !searchQuery.trim()}
                className="bg-gradient-to-r from-purple-600 to-pink-600 text-white"
              >
                {searching ? 'Searching...' : 'Search'}
              </Button>
            </div>

            {/* Search Results */}
            {searchResults.length > 0 && (
              <div className="space-y-2 mt-4">
                <p className="text-sm text-muted-foreground">Search Results:</p>
                {searchResults.map((user) => (
                  <div
                    key={user.id}
                    className="flex items-center justify-between p-3 rounded-lg border border-border/40 bg-background/50 hover:bg-background transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-gradient-to-r from-purple-600 to-pink-600 flex items-center justify-center text-white font-semibold">
                        {user.name?.[0] || user.email[0].toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium">{user.name || 'Unknown User'}</p>
                        <p className="text-sm text-muted-foreground flex items-center gap-1">
                          <Mail className="w-3 h-3" />
                          {user.email}
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      onClick={() => handleAddUser(user.id)}
                      disabled={loading}
                      className="bg-gradient-to-r from-purple-600 to-pink-600 text-white"
                    >
                      <UserPlus className="w-4 h-4 mr-1" />
                      Add
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Access Expiry (Optional) */}
          <div className="space-y-3">
            <label className="text-sm font-semibold flex items-center gap-2">
              <Clock className="w-4 h-4" />
              Access Expiry (Optional)
            </label>
            <Input
              type="datetime-local"
              value={expiryTime}
              onChange={(e) => setExpiryTime(e.target.value)}
              className="w-full"
            />
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <AlertCircle className="w-3 h-3" />
              Leave empty for permanent access
            </p>
          </div>

          {/* Current Access List */}
          <div className="space-y-3">
            <label className="text-sm font-semibold">
              Users with Access ({sharedUsers.length})
            </label>
            {sharedUsers.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <UserMinus className="w-12 h-12 mx-auto mb-2 opacity-50" />
                <p>No users have been granted access yet</p>
              </div>
            ) : (
              <div className="space-y-2">
                {sharedUsers.map((user) => (
                  <div
                    key={user.id}
                    className="flex items-center justify-between p-3 rounded-lg border border-border/40 bg-background/50"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-gradient-to-r from-purple-600 to-pink-600 flex items-center justify-center text-white font-semibold">
                        {user.name?.[0] || user.email[0].toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium">{user.name || 'Unknown User'}</p>
                        <p className="text-sm text-muted-foreground flex items-center gap-1">
                          <Mail className="w-3 h-3" />
                          {user.email}
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleRemoveUser(user.id)}
                      disabled={loading}
                    >
                      <UserMinus className="w-4 h-4 mr-1" />
                      Remove
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 p-6 border-t border-border/40 bg-muted/20">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>
      </Card>
    </div>
  )
}

