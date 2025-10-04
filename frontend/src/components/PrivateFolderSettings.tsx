import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Switch } from '@/components/ui/switch';
import { Lock, Eye, EyeOff, AlertCircle, CheckCircle } from 'lucide-react';

interface PrivateFolderSettingsProps {
  userId: string;
}

export function PrivateFolderSettings({ userId }: PrivateFolderSettingsProps) {
  const [pin, setPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [showPin, setShowPin] = useState(false);
  const [showConfirmPin, setShowConfirmPin] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [hasPIN, setHasPIN] = useState(false);
  const [isPINEnabled, setIsPINEnabled] = useState(false);

  useEffect(() => {
    checkPINStatus();
  }, [userId]);

  const checkPINStatus = async () => {
    try {
      // Check if user has a PIN set by trying to validate with a dummy PIN
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/private-folder/validate-pin`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify({
          user_id: userId,
          pin: 'dummy',
        }),
      });

      const data = await response.json();
      // If we get a specific error about PIN not being set, user doesn't have a PIN
      setHasPIN(data.message !== 'PIN not set. Please set a PIN first.');
    } catch (err) {
      console.error('Error checking PIN status:', err);
    }
  };

  const handleSetPIN = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (pin.length < 4 || pin.length > 8) {
      setError('PIN must be between 4 and 8 characters');
      return;
    }

    if (pin !== confirmPin) {
      setError('PINs do not match');
      return;
    }

    setIsLoading(true);
    setError('');
    setSuccess('');

    try {
      const apiGatewayUrl = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080';
      const response = await fetch(`${apiGatewayUrl}/api/v1/files/private-folder/set-pin`, {
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
        setSuccess('PIN set successfully!');
        setHasPIN(true);
        setPin('');
        setConfirmPin('');
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError('Failed to set PIN. Please try again.');
      console.error('Set PIN error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleTogglePIN = async (enabled: boolean) => {
    if (!enabled && hasPIN) {
      // User wants to disable PIN - they need to enter it to confirm
      const currentPin = prompt('Enter your current PIN to disable private folder protection:');
      if (!currentPin) return;

      // Validate current PIN
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
            pin: currentPin,
          }),
        });

        const data = await response.json();
        if (!data.success) {
          alert('Invalid PIN. Cannot disable private folder protection.');
          return;
        }
      } catch (err) {
        alert('Failed to validate PIN. Please try again.');
        return;
      }
    }

    setIsPINEnabled(enabled);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Lock className="h-5 w-5" />
          Private Folder Settings
        </CardTitle>
        <CardDescription>
          Configure PIN protection for your private folder
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* PIN Status */}
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label className="text-base">Private Folder Protection</Label>
            <div className="text-sm text-muted-foreground">
              {hasPIN ? 'PIN is set and active' : 'No PIN set'}
            </div>
          </div>
          <div className="flex items-center space-x-2">
            {hasPIN ? (
              <CheckCircle className="h-4 w-4 text-green-500" />
            ) : (
              <AlertCircle className="h-4 w-4 text-yellow-500" />
            )}
            <Switch
              checked={isPINEnabled}
              onCheckedChange={handleTogglePIN}
              disabled={!hasPIN}
            />
          </div>
        </div>

        {/* Set PIN Form */}
        {!hasPIN && (
          <form onSubmit={handleSetPIN} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="pin">Set PIN</Label>
              <div className="relative">
                <Input
                  id="pin"
                  type={showPin ? 'text' : 'password'}
                  value={pin}
                  onChange={(e) => setPin(e.target.value)}
                  placeholder="Enter 4-8 digit PIN"
                  maxLength={8}
                  disabled={isLoading}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowPin(!showPin)}
                  disabled={isLoading}
                >
                  {showPin ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPin">Confirm PIN</Label>
              <div className="relative">
                <Input
                  id="confirmPin"
                  type={showConfirmPin ? 'text' : 'password'}
                  value={confirmPin}
                  onChange={(e) => setConfirmPin(e.target.value)}
                  placeholder="Confirm your PIN"
                  maxLength={8}
                  disabled={isLoading}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowConfirmPin(!showConfirmPin)}
                  disabled={isLoading}
                >
                  {showConfirmPin ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {success && (
              <Alert>
                <CheckCircle className="h-4 w-4" />
                <AlertDescription>{success}</AlertDescription>
              </Alert>
            )}

            <Button
              type="submit"
              disabled={isLoading || pin.length < 4 || pin !== confirmPin}
              className="w-full"
            >
              {isLoading ? 'Setting PIN...' : 'Set PIN'}
            </Button>
          </form>
        )}

        {/* PIN Information */}
        <div className="rounded-lg bg-muted p-4">
          <h4 className="font-medium mb-2">PIN Security Features</h4>
          <ul className="text-sm text-muted-foreground space-y-1">
            <li>• PIN must be 4-8 characters long</li>
            <li>• Account locks after 5 failed attempts</li>
            <li>• 15-minute lockout period</li>
            <li>• All access attempts are logged</li>
            <li>• PINs are securely hashed and stored</li>
          </ul>
        </div>
      </CardContent>
    </Card>
  );
}
