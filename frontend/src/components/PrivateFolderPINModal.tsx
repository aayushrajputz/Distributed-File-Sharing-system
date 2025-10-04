import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Lock, Eye, EyeOff, AlertCircle } from 'lucide-react';

interface PrivateFolderPINModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  title?: string;
  description?: string;
  userId: string;
  fileId?: string;
  action: 'access' | 'make-private' | 'remove-private';
}

export function PrivateFolderPINModal({
  isOpen,
  onClose,
  onSuccess,
  title = "Enter PIN",
  description = "Please enter your PIN to access the private folder",
  userId,
  fileId,
  action
}: PrivateFolderPINModalProps) {
  const [pin, setPin] = useState('');
  const [showPin, setShowPin] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [attemptsLeft, setAttemptsLeft] = useState<number | null>(null);
  const [isLocked, setIsLocked] = useState(false);
  const [lockedUntil, setLockedUntil] = useState<string | null>(null);

  // Reset state when modal opens
  useEffect(() => {
    if (isOpen) {
      setPin('');
      setError('');
      setAttemptsLeft(null);
      setIsLocked(false);
      setLockedUntil(null);
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (pin.length < 4) {
      setError('PIN must be at least 4 characters');
      return;
    }

    setIsLoading(true);
    setError('');

    try {
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/private-folder/validate-pin`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify({
          user_id: userId,
          pin: pin,
        }),
      });

      const data = await response.json();

      if (data.success) {
        // PIN is valid, proceed with the action
        if (action === 'make-private' && fileId) {
          await makeFilePrivate();
        } else if (action === 'remove-private' && fileId) {
          await removeFileFromPrivate();
        } else {
          // For access action, just close and call success
          onSuccess();
        }
      } else {
        setError(data.message);
        setAttemptsLeft(data.attempts_left);
        
        if (data.locked_until) {
          setIsLocked(true);
          setLockedUntil(data.locked_until);
        }
      }
    } catch (err) {
      setError('Failed to validate PIN. Please try again.');
      console.error('PIN validation error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const makeFilePrivate = async () => {
    try {
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/private-folder/make-private`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify({
          user_id: userId,
          file_id: fileId,
          pin: pin,
        }),
      });

      const data = await response.json();

      if (data.success) {
        onSuccess();
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError('Failed to make file private. Please try again.');
      console.error('Make private error:', err);
    }
  };

  const removeFileFromPrivate = async () => {
    try {
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/private-folder/remove-from-private`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify({
          user_id: userId,
          file_id: fileId,
          pin: pin,
        }),
      });

      const data = await response.json();

      if (data.success) {
        onSuccess();
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError('Failed to remove file from private folder. Please try again.');
      console.error('Remove from private error:', err);
    }
  };

  const formatLockedUntil = (lockedUntil: string) => {
    try {
      const date = new Date(lockedUntil);
      return date.toLocaleString();
    } catch {
      return lockedUntil;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Lock className="h-5 w-5" />
            {title}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {description}
          </p>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                {error}
                {attemptsLeft !== null && attemptsLeft > 0 && (
                  <span className="block mt-1">
                    {attemptsLeft} attempts remaining
                  </span>
                )}
              </AlertDescription>
            </Alert>
          )}

          {isLocked && lockedUntil && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Account locked until {formatLockedUntil(lockedUntil)}
              </AlertDescription>
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="pin">PIN</Label>
              <div className="relative">
                <Input
                  id="pin"
                  type={showPin ? 'text' : 'password'}
                  value={pin}
                  onChange={(e) => setPin(e.target.value)}
                  placeholder="Enter your PIN"
                  maxLength={8}
                  disabled={isLoading || isLocked}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowPin(!showPin)}
                  disabled={isLoading || isLocked}
                >
                  {showPin ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            <div className="flex justify-end space-x-2">
              <Button
                type="button"
                variant="outline"
                onClick={onClose}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={isLoading || isLocked || pin.length < 4}
              >
                {isLoading ? 'Verifying...' : 'Verify PIN'}
              </Button>
            </div>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
}
