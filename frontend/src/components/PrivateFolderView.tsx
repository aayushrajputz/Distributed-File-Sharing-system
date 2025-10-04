import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { 
  Lock, 
  File, 
  Download, 
  MoreVertical, 
  Search, 
  AlertCircle,
  Calendar,
  HardDrive
} from 'lucide-react';
import { PrivateFolderPINModal } from './PrivateFolderPINModal';
import { FileContextMenu } from './FileContextMenu';

interface PrivateFile {
  file_id: string;
  file_name: string;
  file_size: number;
  content_type: string;
  moved_at: string;
  original_folder?: string;
}

interface PrivateFolderViewProps {
  userId: string;
}

export function PrivateFolderView({ userId }: PrivateFolderViewProps) {
  const [files, setFiles] = useState<PrivateFile[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [totalFiles, setTotalFiles] = useState(0);
  const [currentPage, setCurrentPage] = useState(0);
  const [showPINModal, setShowPINModal] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [totalSize, setTotalSize] = useState(0);

  const filesPerPage = 10;

  useEffect(() => {
    if (isAuthenticated) {
      loadPrivateFiles();
    }
  }, [userId, isAuthenticated, currentPage]);

  const loadPrivateFiles = async () => {
    setIsLoading(true);
    setError('');

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8080'}/api/v1/files/private-folder/files?user_id=${userId}&limit=${filesPerPage}&offset=${currentPage * filesPerPage}`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
          },
        }
      );

      const data = await response.json();

      if (data.success) {
        setFiles(data.files);
        setTotalFiles(data.total);
        
        // Calculate total size
        const size = data.files.reduce((sum: number, file: PrivateFile) => sum + file.file_size, 0);
        setTotalSize(size);
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError('Failed to load private files');
      console.error('Load private files error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAccessPrivateFolder = () => {
    setShowPINModal(true);
  };

  const handlePINSuccess = () => {
    setShowPINModal(false);
    setIsAuthenticated(true);
  };

  const handlePINClose = () => {
    setShowPINModal(false);
  };

  const handleDownload = (fileId: string) => {
    // Implement download logic
    console.log('Download file:', fileId);
  };

  const handleShare = (fileId: string) => {
    // Implement share logic
    console.log('Share file:', fileId);
  };

  const handleDelete = (fileId: string) => {
    // Implement delete logic
    console.log('Delete file:', fileId);
  };

  const handlePrivacyChange = (fileId: string, isPrivate: boolean) => {
    // Remove file from the list if it's no longer private
    if (!isPrivate) {
      setFiles(files.filter(file => file.file_id !== fileId));
      setTotalFiles(totalFiles - 1);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const filteredFiles = files.filter(file =>
    file.file_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const totalPages = Math.ceil(totalFiles / filesPerPage);

  if (!isAuthenticated) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lock className="h-5 w-5" />
            Private Folder
          </CardTitle>
          <CardDescription>
            Enter your PIN to access your private files
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center space-y-4">
            <div className="mx-auto w-12 h-12 bg-muted rounded-full flex items-center justify-center">
              <Lock className="h-6 w-6" />
            </div>
            <div>
              <h3 className="text-lg font-medium">Private Folder Protected</h3>
              <p className="text-muted-foreground">
                Your private files are secured with PIN protection
              </p>
            </div>
            <Button onClick={handleAccessPrivateFolder} className="w-full">
              Enter PIN to Access
            </Button>
          </div>

          <PrivateFolderPINModal
            isOpen={showPINModal}
            onClose={handlePINClose}
            onSuccess={handlePINSuccess}
            title="Access Private Folder"
            description="Enter your PIN to access your private files"
            userId={userId}
            action="access"
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Lock className="h-5 w-5" />
          Private Folder
        </CardTitle>
        <CardDescription>
          Your private files are protected with PIN authentication
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="flex items-center space-x-2">
            <File className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">
              {totalFiles} files
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <HardDrive className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">
              {formatFileSize(totalSize)}
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">
              Last accessed: {new Date().toLocaleDateString()}
            </span>
          </div>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search private files..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-10"
          />
        </div>

        {/* Files List */}
        {isLoading ? (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
            <p className="text-sm text-muted-foreground mt-2">Loading private files...</p>
          </div>
        ) : error ? (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : filteredFiles.length === 0 ? (
          <div className="text-center py-8">
            <File className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium">No private files</h3>
            <p className="text-muted-foreground">
              {searchTerm ? 'No files match your search' : 'Your private folder is empty'}
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredFiles.map((file) => (
              <div
                key={file.file_id}
                className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50"
              >
                <div className="flex items-center space-x-3">
                  <File className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="font-medium">{file.file_name}</p>
                    <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                      <span>{formatFileSize(file.file_size)}</span>
                      <span>•</span>
                      <span>{formatDate(file.moved_at)}</span>
                      <Badge variant="secondary" className="text-xs">
                        Private
                      </Badge>
                    </div>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDownload(file.file_id)}
                  >
                    <Download className="h-4 w-4" />
                  </Button>
                  <FileContextMenu
                    fileId={file.file_id}
                    fileName={file.file_name}
                    isPrivate={true}
                    userId={userId}
                    onDownload={() => handleDownload(file.file_id)}
                    onShare={() => handleShare(file.file_id)}
                    onDelete={() => handleDelete(file.file_id)}
                    onPrivacyChange={handlePrivacyChange}
                  />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex justify-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
              disabled={currentPage === 0}
            >
              Previous
            </Button>
            <span className="flex items-center px-3 text-sm text-muted-foreground">
              Page {currentPage + 1} of {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage(Math.min(totalPages - 1, currentPage + 1))}
              disabled={currentPage >= totalPages - 1}
            >
              Next
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
