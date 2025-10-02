'use client'

import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  X, 
  Copy, 
  Check, 
  Calendar, 
  Users, 
  Shield, 
  Clock,
  Link as LinkIcon,
  Eye,
  Download,
  Edit,
  AlertCircle,
  CheckCircle
} from 'lucide-react'
import { fileService } from '@/lib/api/files'
import { useAuthStore } from '@/store/auth'
import { validateEmailList, formatEmailList, isEmailListEmpty, EmailListValidationResult } from '@/lib/utils/validation'

interface FileSharingModalProps {
  fileId: string
  fileName: string
  onClose: () => void
}

type Permission = 'READ' | 'WRITE' | 'ADMIN'
type ExpiryOption = '1h' | '1d' | '7d' | '30d' | 'never'

const permissionLabels = {
  READ: 'View only',
  WRITE: 'View and edit',
  ADMIN: 'Full access'
}

const permissionIcons = {
  READ: <Eye className="w-4 h-4" />,
  WRITE: <Edit className="w-4 h-4" />,
  ADMIN: <Shield className="w-4 h-4" />
}

const expiryOptions = [
  { value: '1h', label: '1 hour' },
  { value: '1d', label: '1 day' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: 'never', label: 'Never' }
]

export function FileSharingModal({ fileId, fileName, onClose }: FileSharingModalProps) {
  const { user } = useAuthStore()
  const [emails, setEmails] = useState('')
  const [permission, setPermission] = useState<Permission>('READ')
  const [expiry, setExpiry] = useState<ExpiryOption>('7d')
  const [shareLink, setShareLink] = useState('')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  
  // Real-time validation states
  const [emailValidation, setEmailValidation] = useState<EmailListValidationResult>({
    isValid: true,
    validEmails: [],
    invalidEmails: [],
    errors: []
  })
  const [isFormValid, setIsFormValid] = useState(false)
  const [showEmailValidation, setShowEmailValidation] = useState(false)

  // Real-time email validation
  const validateEmails = useCallback((emailList: string) => {
    const validation = validateEmailList(emailList)
    setEmailValidation(validation)
    return validation
  }, [])

  // Handle email input change with real-time validation
  const handleEmailChange = (value: string) => {
    setEmails(value)
    const validation = validateEmails(value)
    setShowEmailValidation(value.length > 0)
  }

  // Generate share link with proper format
  const generateShareLink = () => {
    const baseUrl = process.env.NEXT_PUBLIC_FRONTEND_URL || window.location.origin
    const link = `${baseUrl}/shared/${fileId}`
    setShareLink(link)
  }

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(shareLink)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  // Form validation effect
  useEffect(() => {
    const isValid = 
      emailValidation.isValid && 
      (isEmailListEmpty(emails) || emailValidation.validEmails.length > 0) &&
      !!permission &&
      !!expiry

    setIsFormValid(isValid)
  }, [emailValidation, emails, permission, expiry])

  const handleShare = async () => {
    if (!user) {
      setError('You must be logged in to share files')
      return
    }

    // Validate form before submission
    if (!isFormValid) {
      setError('Please fix the form errors before sharing')
      return
    }

    // Allow sharing with link only (no emails) or with valid emails
    if (!isEmailListEmpty(emails) && !emailValidation.isValid) {
      setError('Please fix email validation errors before sharing')
      return
    }

    setLoading(true)
    setError('')

    try {
      const emailList = isEmailListEmpty(emails) ? [] : emailValidation.validEmails
      const expiryTime = expiry === 'never' ? null : getExpiryTimestamp()
      
      await fileService.shareFile(fileId, emailList, permission, expiryTime)
      
      // Generate share link
      generateShareLink()
      
      // Show success message
      setError('')
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to share file')
    } finally {
      setLoading(false)
    }
  }

  // Get expiry timestamp for backend
  const getExpiryTimestamp = (): string | null => {
    if (expiry === 'never') return null
    
    const now = new Date()
    const hours = {
      '1h': 1,
      '1d': 24,
      '7d': 24 * 7,
      '30d': 24 * 30
    }[expiry] || 24
    
    const expiryDate = new Date(now.getTime() + hours * 60 * 60 * 1000)
    return expiryDate.toISOString()
  }

  const getExpiryDate = () => {
    if (expiry === 'never') return 'Never expires'
    
    const now = new Date()
    const hours = {
      '1h': 1,
      '1d': 24,
      '7d': 24 * 7,
      '30d': 24 * 30
    }[expiry] || 24
    
    const expiryDate = new Date(now.getTime() + hours * 60 * 60 * 1000)
    return expiryDate.toLocaleString()
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <Card className="w-full max-w-md animate-scale-in">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <div>
            <CardTitle className="text-lg font-semibold">Share File</CardTitle>
            <CardDescription className="text-sm text-muted-foreground truncate">
              {fileName}
            </CardDescription>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </CardHeader>

        <CardContent className="space-y-6">
          {error && (
            <div className="notification-error rounded-lg p-3 text-sm animate-slide-down">
              {error}
            </div>
          )}

          {/* Email addresses */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Share with</label>
            <div className="relative">
              <Input
                placeholder="Enter email addresses (comma-separated) or leave empty for link-only sharing"
                value={emails}
                onChange={(e) => handleEmailChange(e.target.value)}
                leftIcon={<Users className="w-4 h-4" />}
                className={`${
                  showEmailValidation && !emailValidation.isValid 
                    ? 'border-red-500 focus:border-red-500' 
                    : showEmailValidation && emailValidation.isValid 
                    ? 'border-green-500 focus:border-green-500' 
                    : ''
                }`}
              />
              {showEmailValidation && (
                <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                  {emailValidation.isValid ? (
                    <CheckCircle className="w-4 h-4 text-green-500" />
                  ) : (
                    <AlertCircle className="w-4 h-4 text-red-500" />
                  )}
                </div>
              )}
            </div>
            
            {/* Email validation feedback */}
            {showEmailValidation && (
              <div className="space-y-1">
                {emailValidation.isValid ? (
                  <p className="text-xs text-green-600 flex items-center gap-1">
                    <CheckCircle className="w-3 h-3" />
                    {emailValidation.validEmails.length} valid email{emailValidation.validEmails.length !== 1 ? 's' : ''}
                  </p>
                ) : (
                  <div className="space-y-1">
                    {emailValidation.errors.map((error, index) => (
                      <p key={index} className="text-xs text-red-600 flex items-center gap-1">
                        <AlertCircle className="w-3 h-3" />
                        {error}
                      </p>
                    ))}
                  </div>
                )}
              </div>
            )}
            
            <p className="text-xs text-muted-foreground">
              Separate multiple emails with commas, or leave empty for link-only sharing
            </p>
          </div>

          {/* Permission level */}
          <div className="space-y-3">
            <label className="text-sm font-medium">Permission level</label>
            <div className="grid grid-cols-3 gap-2">
              {Object.entries(permissionLabels).map(([key, label]) => (
                <Button
                  key={key}
                  variant={permission === key ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setPermission(key as Permission)}
                  className="flex flex-col items-center space-y-1 h-auto py-3"
                >
                  {permissionIcons[key as Permission]}
                  <span className="text-xs">{label}</span>
                </Button>
              ))}
            </div>
          </div>

          {/* Expiry */}
          <div className="space-y-3">
            <label className="text-sm font-medium">Expires</label>
            <div className="grid grid-cols-2 gap-2">
              {expiryOptions.map((option) => (
                <Button
                  key={option.value}
                  variant={expiry === option.value ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setExpiry(option.value as ExpiryOption)}
                  className="justify-start"
                >
                  <Clock className="w-4 h-4 mr-2" />
                  {option.label}
                </Button>
              ))}
            </div>
            {expiry !== 'never' && (
              <p className="text-xs text-muted-foreground">
                Expires: {getExpiryDate()}
              </p>
            )}
          </div>

          {/* Share link */}
          {shareLink && (
            <div className="space-y-2">
              <label className="text-sm font-medium">Share link</label>
              <div className="flex space-x-2">
                <Input
                  value={shareLink}
                  readOnly
                  leftIcon={<LinkIcon className="w-4 h-4" />}
                  className="flex-1"
                />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={copyToClipboard}
                  className="shrink-0"
                >
                  {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Anyone with this link can access the file
              </p>
            </div>
          )}

          {/* Actions */}
          <div className="flex space-x-2 pt-4">
            <Button
              variant="outline"
              onClick={onClose}
              className="flex-1"
            >
              Cancel
            </Button>
            <Button
              onClick={handleShare}
              loading={loading}
              disabled={!isFormValid || loading}
              className="flex-1"
            >
              {shareLink ? 'Update Share' : 'Share File'}
            </Button>
          </div>

          {/* Quick share link generation */}
          {!shareLink && (
            <div className="pt-2 border-t">
              <Button
                variant="ghost"
                onClick={generateShareLink}
                className="w-full"
                leftIcon={<LinkIcon className="w-4 h-4" />}
              >
                Generate share link
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
