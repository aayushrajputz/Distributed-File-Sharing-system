'use client';

import React, { useState, useCallback } from 'react';
import { X, Upload, Lock, Globe, AlertCircle, CheckCircle2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface UploadWithPrivacyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpload: (files: FileList, isPrivate: boolean) => Promise<void>;
  disabled?: boolean;
}

export default function UploadWithPrivacyModal({
  isOpen,
  onClose,
  onUpload,
  disabled = false,
}: UploadWithPrivacyModalProps) {
  const [isDragging, setIsDragging] = useState(false);
  const [isPrivate, setIsPrivate] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<File[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!disabled && !isUploading) {
      setIsDragging(true);
    }
  }, [disabled, isUploading]);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    if (disabled || isUploading) return;

    const files = Array.from(e.dataTransfer.files);
    setSelectedFiles(files);
    setError(null);
  }, [disabled, isUploading]);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (disabled || isUploading || !e.target.files) return;

    const files = Array.from(e.target.files);
    setSelectedFiles(files);
    setError(null);
  };

  const handleUpload = async () => {
    if (selectedFiles.length === 0) {
      setError('Please select files to upload');
      return;
    }

    setIsUploading(true);
    setError(null);

    try {
      const fileList = new DataTransfer();
      selectedFiles.forEach(file => fileList.items.add(file));
      
      await onUpload(fileList.files, isPrivate);
      
      // Reset state
      setSelectedFiles([]);
      setIsPrivate(false);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to upload files');
      console.error('Upload error:', err);
    } finally {
      setIsUploading(false);
    }
  };

  const removeFile = (index: number) => {
    setSelectedFiles(prev => prev.filter((_, i) => i !== index));
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="bg-gradient-to-r from-blue-600 to-indigo-600 p-6 text-white">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-bold">Upload Files</h2>
              <p className="text-blue-100 mt-1 text-sm">Choose files and set privacy options</p>
            </div>
            <button
              onClick={onClose}
              disabled={isUploading}
              className="p-2 hover:bg-white/20 rounded-lg transition-colors disabled:opacity-50"
            >
              <X className="w-6 h-6" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-250px)]">
          {/* Error Message */}
          {error && (
            <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
              <p className="text-red-800 dark:text-red-200 text-sm">{error}</p>
            </div>
          )}

          {/* Privacy Toggle */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Privacy Settings
            </label>
            <div className="grid grid-cols-2 gap-4">
              <button
                type="button"
                onClick={() => setIsPrivate(false)}
                disabled={isUploading}
                className={cn(
                  'p-4 rounded-lg border-2 transition-all',
                  !isPrivate
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600',
                  isUploading && 'opacity-50 cursor-not-allowed'
                )}
              >
                <div className="flex items-center gap-3">
                  <div className={cn(
                    'w-10 h-10 rounded-full flex items-center justify-center',
                    !isPrivate ? 'bg-blue-500 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                  )}>
                    <Globe className="w-5 h-5" />
                  </div>
                  <div className="text-left">
                    <p className="font-semibold text-gray-900 dark:text-white">Public</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">Anyone with link can access</p>
                  </div>
                </div>
              </button>

              <button
                type="button"
                onClick={() => setIsPrivate(true)}
                disabled={isUploading}
                className={cn(
                  'p-4 rounded-lg border-2 transition-all',
                  isPrivate
                    ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600',
                  isUploading && 'opacity-50 cursor-not-allowed'
                )}
              >
                <div className="flex items-center gap-3">
                  <div className={cn(
                    'w-10 h-10 rounded-full flex items-center justify-center',
                    isPrivate ? 'bg-purple-500 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                  )}>
                    <Lock className="w-5 h-5" />
                  </div>
                  <div className="text-left">
                    <p className="font-semibold text-gray-900 dark:text-white">Private</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">Only you and invited users</p>
                  </div>
                </div>
              </button>
            </div>
          </div>

          {/* File Drop Zone */}
          <div
            className={cn(
              'border-2 border-dashed rounded-lg p-8 text-center transition-all cursor-pointer',
              isDragging && 'border-blue-500 bg-blue-50 dark:bg-blue-900/20',
              !isDragging && 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500',
              (disabled || isUploading) && 'opacity-50 cursor-not-allowed'
            )}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            onClick={() => !disabled && !isUploading && document.getElementById('file-upload-input')?.click()}
          >
            <input
              id="file-upload-input"
              type="file"
              multiple
              onChange={handleFileSelect}
              className="hidden"
              disabled={disabled || isUploading}
            />

            <div className="space-y-4">
              <div className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-500 rounded-full flex items-center justify-center">
                <Upload className="w-8 h-8 text-white" />
              </div>

              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {isDragging ? 'Drop files here' : 'Drag and drop files'}
                </h3>
                <p className="text-gray-500 dark:text-gray-400 mt-1">
                  or click to browse your computer
                </p>
              </div>
            </div>
          </div>

          {/* Selected Files List */}
          {selectedFiles.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Selected Files ({selectedFiles.length})
              </h3>
              <div className="space-y-2 max-h-48 overflow-y-auto">
                {selectedFiles.map((file, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                  >
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      <CheckCircle2 className="w-5 h-5 text-green-500 flex-shrink-0" />
                      <div className="min-w-0 flex-1">
                        <p className="font-medium text-gray-900 dark:text-white truncate">{file.name}</p>
                        <p className="text-sm text-gray-500 dark:text-gray-400">{formatFileSize(file.size)}</p>
                      </div>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        removeFile(index);
                      }}
                      disabled={isUploading}
                      className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors disabled:opacity-50"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-800/50 flex gap-3">
          <button
            onClick={onClose}
            disabled={isUploading}
            className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleUpload}
            disabled={isUploading || selectedFiles.length === 0}
            className="flex-1 px-4 py-2 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-lg hover:from-blue-700 hover:to-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {isUploading ? (
              <>
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Uploading...
              </>
            ) : (
              <>
                <Upload className="w-4 h-4" />
                Upload {selectedFiles.length > 0 && `(${selectedFiles.length})`}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

