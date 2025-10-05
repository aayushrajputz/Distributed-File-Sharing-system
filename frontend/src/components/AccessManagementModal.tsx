'use client';

import React, { useState, useEffect } from 'react';
import { X, UserPlus, Trash2, Search, Mail, Clock, AlertCircle } from 'lucide-react';
import { fileService } from '@/lib/api/files';

interface AccessManagementModalProps {
  isOpen: boolean;
  onClose: () => void;
  file: {
    id: string;
    name: string;
    shared_with?: string[];
  };
  onAccessUpdated?: () => void;
}

interface User {
  id: string;
  email: string;
  name?: string;
}

export default function AccessManagementModal({
  isOpen,
  onClose,
  file,
  onAccessUpdated,
}: AccessManagementModalProps) {
  const [sharedWith, setSharedWith] = useState<string[]>(file.shared_with || []);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<User[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  useEffect(() => {
    setSharedWith(file.shared_with || []);
  }, [file.shared_with]);

  // Search users by email
  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      setSearchResults([]);
      return;
    }

    setIsSearching(true);
    setError(null);

    try {
      // Mock user search - replace with actual API call
      // const response = await fetch(`/api/v1/users/search?q=${encodeURIComponent(searchQuery)}`);
      // const data = await response.json();
      
      // For now, simulate search results
      const mockUsers: User[] = [
        { id: 'user1', email: searchQuery, name: 'User One' },
        { id: 'user2', email: `${searchQuery}.test`, name: 'Test User' },
      ];
      
      setSearchResults(mockUsers.filter(u => !sharedWith.includes(u.email)));
    } catch (err) {
      setError('Failed to search users');
      console.error('User search error:', err);
    } finally {
      setIsSearching(false);
    }
  };

  // Add user to private access
  const handleAddUser = async (userEmail: string) => {
    setIsLoading(true);
    setError(null);
    setSuccessMessage(null);

    try {
      await fileService.managePrivateAccess(file.id, [userEmail], 'add');
      setSharedWith([...sharedWith, userEmail]);
      setSuccessMessage(`Added ${userEmail} to private access`);
      setSearchQuery('');
      setSearchResults([]);
      
      if (onAccessUpdated) {
        onAccessUpdated();
      }
    } catch (err: any) {
      setError(err.message || 'Failed to add user');
      console.error('Add user error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Remove user from private access
  const handleRemoveUser = async (userEmail: string) => {
    setIsLoading(true);
    setError(null);
    setSuccessMessage(null);

    try {
      await fileService.managePrivateAccess(file.id, [userEmail], 'remove');
      setSharedWith(sharedWith.filter(email => email !== userEmail));
      setSuccessMessage(`Removed ${userEmail} from private access`);
      
      if (onAccessUpdated) {
        onAccessUpdated();
      }
    } catch (err: any) {
      setError(err.message || 'Failed to remove user');
      console.error('Remove user error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="bg-gradient-to-r from-purple-600 to-pink-600 p-6 text-white">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-bold">Manage Access</h2>
              <p className="text-purple-100 mt-1 text-sm truncate max-w-md">{file.name}</p>
            </div>
            <button
              onClick={onClose}
              className="p-2 hover:bg-white/20 rounded-lg transition-colors"
            >
              <X className="w-6 h-6" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          {/* Error/Success Messages */}
          {error && (
            <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
              <p className="text-red-800 dark:text-red-200 text-sm">{error}</p>
            </div>
          )}

          {successMessage && (
            <div className="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
              <p className="text-green-800 dark:text-green-200 text-sm">{successMessage}</p>
            </div>
          )}

          {/* User Search */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Add User by Email
            </label>
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="email"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                  placeholder="Enter user email..."
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent dark:bg-gray-700 dark:text-white"
                />
              </div>
              <button
                onClick={handleSearch}
                disabled={isSearching || !searchQuery.trim()}
                className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                <Search className="w-5 h-5" />
                Search
              </button>
            </div>

            {/* Search Results */}
            {searchResults.length > 0 && (
              <div className="mt-3 border border-gray-200 dark:border-gray-700 rounded-lg divide-y divide-gray-200 dark:divide-gray-700">
                {searchResults.map((user) => (
                  <div
                    key={user.id}
                    className="p-3 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50"
                  >
                    <div>
                      <p className="font-medium text-gray-900 dark:text-white">{user.email}</p>
                      {user.name && (
                        <p className="text-sm text-gray-500 dark:text-gray-400">{user.name}</p>
                      )}
                    </div>
                    <button
                      onClick={() => handleAddUser(user.email)}
                      disabled={isLoading}
                      className="px-3 py-1.5 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 flex items-center gap-2 text-sm"
                    >
                      <UserPlus className="w-4 h-4" />
                      Add
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Current Access List */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
              Users with Access ({sharedWith.length})
            </h3>

            {sharedWith.length === 0 ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                <Mail className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>No users have been granted access yet</p>
                <p className="text-sm mt-1">Search and add users above</p>
              </div>
            ) : (
              <div className="space-y-2">
                {sharedWith.map((email) => (
                  <div
                    key={email}
                    className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg flex items-center justify-between"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full flex items-center justify-center text-white font-semibold">
                        {email.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium text-gray-900 dark:text-white">{email}</p>
                        <p className="text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          Full access
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRemoveUser(email)}
                      disabled={isLoading}
                      className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors disabled:opacity-50"
                      title="Remove access"
                    >
                      <Trash2 className="w-5 h-5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-800/50">
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

