import React, { useState } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { MoreVertical, Download, Share, Trash2, Lock, Unlock, Eye } from 'lucide-react';
import { PrivateFolderPINModal } from './PrivateFolderPINModal';

interface FileContextMenuProps {
  fileId: string;
  fileName: string;
  isPrivate: boolean;
  userId: string;
  onDownload: () => void;
  onShare: () => void;
  onDelete: () => void;
  onPrivacyChange: (fileId: string, isPrivate: boolean) => void;
}

export function FileContextMenu({
  fileId,
  fileName,
  isPrivate,
  userId,
  onDownload,
  onShare,
  onDelete,
  onPrivacyChange,
}: FileContextMenuProps) {
  const [showPINModal, setShowPINModal] = useState(false);
  const [pinAction, setPinAction] = useState<'make-private' | 'remove-private'>('make-private');

  const handleMakePrivate = () => {
    setPinAction('make-private');
    setShowPINModal(true);
  };

  const handleRemoveFromPrivate = () => {
    setPinAction('remove-private');
    setShowPINModal(true);
  };

  const handlePINSuccess = () => {
    setShowPINModal(false);
    onPrivacyChange(fileId, pinAction === 'make-private');
  };

  const handlePINClose = () => {
    setShowPINModal(false);
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
            <MoreVertical className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onClick={onDownload}>
            <Download className="mr-2 h-4 w-4" />
            Download
          </DropdownMenuItem>
          
          <DropdownMenuItem onClick={onShare}>
            <Share className="mr-2 h-4 w-4" />
            Share
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          {isPrivate ? (
            <DropdownMenuItem onClick={handleRemoveFromPrivate}>
              <Unlock className="mr-2 h-4 w-4" />
              Remove from Private
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={handleMakePrivate}>
              <Lock className="mr-2 h-4 w-4" />
              Make Private
            </DropdownMenuItem>
          )}

          <DropdownMenuSeparator />

          <DropdownMenuItem onClick={onDelete} className="text-destructive">
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <PrivateFolderPINModal
        isOpen={showPINModal}
        onClose={handlePINClose}
        onSuccess={handlePINSuccess}
        title={isPrivate ? "Remove from Private Folder" : "Make File Private"}
        description={
          isPrivate
            ? `Enter your PIN to remove "${fileName}" from the private folder`
            : `Enter your PIN to move "${fileName}" to the private folder`
        }
        userId={userId}
        fileId={fileId}
        action={pinAction}
      />
    </>
  );
}

