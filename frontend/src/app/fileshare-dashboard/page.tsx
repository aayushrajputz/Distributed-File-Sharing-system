'use client';

import React, { useState } from 'react';
import { 
  Upload, Search, Bell, Moon, Sun, User, Menu, X,
  Home, File, Users, Star, Trash2, Settings, Filter,
  Grid, List, Download, Edit, Share2, MoreVertical,
  FolderOpen, Image, FileText, Music, Video, Archive,
  ChevronDown, Plus, Check, AlertCircle, Info
} from 'lucide-react';

// Types
interface FileItem {
  id: string;
  name: string;
  type: 'image' | 'document' | 'video' | 'audio' | 'archive' | 'other';
  size: string;
  date: string;
  thumbnail?: string;
  shared: boolean;
  favorite: boolean;
}

interface UploadingFile {
  name: string;
  progress: number;
  size: string;
}

type ViewMode = 'grid' | 'list';
type FilterType = 'all' | 'image' | 'document' | 'video' | 'audio' | 'archive';
type SortOption = 'name' | 'date' | 'size';

const FileShareDashboard = () => {
  // State Management
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [darkMode, setDarkMode] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [filterType, setFilterType] = useState<FilterType>('all');
  const [sortBy, setSortBy] = useState<SortOption>('date');
  const [searchQuery, setSearchQuery] = useState('');
  const [isDragging, setIsDragging] = useState(false);
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([]);
  const [showNotifications, setShowNotifications] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [activeNav, setActiveNav] = useState('dashboard');

  // Sample Data
  const files: FileItem[] = [
    { id: '1', name: 'Project Proposal.pdf', type: 'document', size: '2.4 MB', date: '2025-09-28', shared: true, favorite: true },
    { id: '2', name: 'Design Mockup.png', type: 'image', size: '5.1 MB', date: '2025-09-27', shared: false, favorite: false },
    { id: '3', name: 'Presentation.mp4', type: 'video', size: '45.2 MB', date: '2025-09-26', shared: true, favorite: true },
    { id: '4', name: 'Meeting Notes.docx', type: 'document', size: '156 KB', date: '2025-09-25', shared: false, favorite: false },
    { id: '5', name: 'Background Music.mp3', type: 'audio', size: '8.3 MB', date: '2025-09-24', shared: false, favorite: false },
    { id: '6', name: 'Archive.zip', type: 'archive', size: '120 MB', date: '2025-09-23', shared: true, favorite: false },
  ];

  // Navigation Items
  const navItems = [
    { id: 'dashboard', icon: Home, label: 'Dashboard' },
    { id: 'myfiles', icon: File, label: 'My Files' },
    { id: 'shared', icon: Users, label: 'Shared with Me' },
    { id: 'favorites', icon: Star, label: 'Favorites' },
    { id: 'trash', icon: Trash2, label: 'Trash' },
    { id: 'settings', icon: Settings, label: 'Settings' },
  ];

  // File Type Icons
  const getFileIcon = (type: string) => {
    const icons = {
      image: Image,
      document: FileText,
      video: Video,
      audio: Music,
      archive: Archive,
      other: File,
    };
    return icons[type as keyof typeof icons] || File;
  };

  // Handle File Upload
  const handleFileUpload = (files: FileList | null) => {
    if (!files) return;
    
    const newFiles: UploadingFile[] = Array.from(files).map(file => ({
      name: file.name,
      progress: 0,
      size: `${(file.size / 1024 / 1024).toFixed(2)} MB`,
    }));

    setUploadingFiles(prev => [...prev, ...newFiles]);

    // Simulate upload progress
    newFiles.forEach((file, index) => {
      const interval = setInterval(() => {
        setUploadingFiles(prev => {
          const updated = [...prev];
          const fileIndex = prev.findIndex(f => f.name === file.name);
          if (fileIndex !== -1 && updated[fileIndex].progress < 100) {
            updated[fileIndex].progress += 10;
          } else {
            clearInterval(interval);
          }
          return updated;
        });
      }, 300);
    });
  };

  return (
    <div className={`min-h-screen ${darkMode ? 'dark bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900' : 'bg-gradient-to-br from-gray-50 via-white to-gray-100'} font-['Inter',sans-serif] transition-colors duration-300`}>
      
      {/* Header */}
      <header className="sticky top-0 z-50 backdrop-blur-xl bg-white/5 dark:bg-gray-900/50 border-b border-white/10">
        <div className="flex items-center justify-between px-6 py-4">
          {/* Left Section */}
          <div className="flex items-center gap-4">
            <button 
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="lg:hidden p-2 rounded-xl hover:bg-white/10 transition-all"
            >
              {sidebarOpen ? <X size={24} /> : <Menu size={24} />}
            </button>
            
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-lg shadow-blue-500/50">
                <FolderOpen className="text-white" size={24} />
              </div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
                FileShare
              </h1>
            </div>
          </div>

          {/* Center - Search */}
          <div className="hidden md:flex flex-1 max-w-2xl mx-8">
            <div className="relative w-full group">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 group-hover:text-blue-400 transition-colors" size={20} />
              <input
                type="text"
                placeholder="Search files, folders, or tags..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-12 pr-4 py-3 rounded-2xl bg-white/5 border border-white/10 focus:border-blue-500/50 focus:bg-white/10 outline-none text-white placeholder-gray-400 transition-all backdrop-blur-xl"
              />
            </div>
          </div>

          {/* Right Section */}
          <div className="flex items-center gap-3">
            {/* Dark Mode Toggle */}
            <button
              onClick={() => setDarkMode(!darkMode)}
              className="p-3 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 transition-all hover:scale-105"
            >
              {darkMode ? <Sun size={20} className="text-yellow-400" /> : <Moon size={20} className="text-blue-500" />}
            </button>

            {/* Notifications */}
            <div className="relative">
              <button
                onClick={() => setShowNotifications(!showNotifications)}
                className="p-3 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 transition-all hover:scale-105 relative"
              >
                <Bell size={20} className="text-gray-300" />
                <span className="absolute top-2 right-2 w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
              </button>
            </div>

            {/* User Profile */}
            <div className="relative">
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="flex items-center gap-3 p-2 pr-4 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 transition-all hover:scale-105"
              >
                <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                  <User size={18} className="text-white" />
                </div>
                <span className="hidden lg:block text-sm font-medium text-gray-300">John Doe</span>
                <ChevronDown size={16} className="text-gray-400" />
              </button>
            </div>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar */}
        <aside className={`${sidebarOpen ? 'translate-x-0' : '-translate-x-full'} lg:translate-x-0 fixed lg:sticky top-[73px] left-0 h-[calc(100vh-73px)] w-64 backdrop-blur-xl bg-white/5 dark:bg-gray-900/50 border-r border-white/10 transition-transform duration-300 z-40 overflow-y-auto`}>
          <nav className="p-4 space-y-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <button
                  key={item.id}
                  onClick={() => setActiveNav(item.id)}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
                    activeNav === item.id
                      ? 'bg-gradient-to-r from-blue-500/20 to-purple-500/20 border border-blue-500/50 text-blue-400 shadow-lg shadow-blue-500/20'
                      : 'hover:bg-white/5 text-gray-400 hover:text-gray-200'
                  }`}
                >
                  <Icon size={20} />
                  <span className="font-medium">{item.label}</span>
                </button>
              );
            })}
          </nav>

          {/* Storage Info */}
          <div className="m-4 p-4 rounded-2xl bg-gradient-to-br from-blue-500/10 to-purple-500/10 border border-white/10">
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm text-gray-400">Storage</span>
              <span className="text-sm font-semibold text-blue-400">75%</span>
            </div>
            <div className="w-full h-2 bg-gray-700/50 rounded-full overflow-hidden">
              <div className="h-full w-3/4 bg-gradient-to-r from-blue-500 to-purple-600 rounded-full"></div>
            </div>
            <p className="text-xs text-gray-500 mt-2">7.5 GB of 10 GB used</p>
            <button className="mt-3 w-full py-2 rounded-lg bg-gradient-to-r from-blue-500 to-purple-600 text-white text-sm font-medium hover:shadow-lg hover:shadow-blue-500/50 transition-all">
              Upgrade Plan
            </button>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-6 lg:p-8 overflow-y-auto">
          {/* Alert Banners */}
          <div className="space-y-3 mb-6">
            <div className="flex items-center gap-3 p-4 rounded-2xl bg-gradient-to-r from-blue-500/10 to-blue-600/10 border border-blue-500/30 backdrop-blur-xl">
              <Info className="text-blue-400" size={20} />
              <p className="text-sm text-blue-300">Your storage is 75% full. Consider upgrading your plan.</p>
              <button className="ml-auto text-blue-400 hover:text-blue-300 transition-colors">
                <X size={18} />
              </button>
            </div>
          </div>

          {/* Upload Section */}
          <div className="mb-8">
            <div
              onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
              onDragLeave={() => setIsDragging(false)}
              onDrop={(e) => {
                e.preventDefault();
                setIsDragging(false);
                handleFileUpload(e.dataTransfer.files);
              }}
              className={`relative rounded-3xl border-2 border-dashed transition-all ${
                isDragging
                  ? 'border-blue-500 bg-blue-500/10 scale-[1.02]'
                  : 'border-white/20 bg-white/5 hover:bg-white/10'
              } backdrop-blur-xl p-12 text-center group cursor-pointer`}
            >
              <input
                type="file"
                multiple
                onChange={(e) => handleFileUpload(e.target.files)}
                className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              />
              <div className="flex flex-col items-center gap-4">
                <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                  <Upload className="text-blue-400" size={32} />
                </div>
                <div>
                  <h3 className="text-xl font-semibold text-white mb-2">Drop files here or click to upload</h3>
                  <p className="text-gray-400 text-sm">Support for multiple files • Max 100MB per file</p>
                </div>
                <button className="px-6 py-3 rounded-xl bg-gradient-to-r from-blue-500 to-purple-600 text-white font-medium hover:shadow-lg hover:shadow-blue-500/50 transition-all flex items-center gap-2">
                  <Plus size={20} />
                  Choose Files
                </button>
              </div>
            </div>

            {/* Upload Progress */}
            {uploadingFiles.length > 0 && (
              <div className="mt-4 space-y-3">
                {uploadingFiles.map((file, index) => (
                  <div key={index} className="p-4 rounded-2xl bg-white/5 backdrop-blur-xl border border-white/10">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-3">
                        {file.progress === 100 ? (
                          <Check className="text-green-400" size={20} />
                        ) : (
                          <Upload className="text-blue-400 animate-pulse" size={20} />
                        )}
                        <span className="text-sm font-medium text-white">{file.name}</span>
                      </div>
                      <span className="text-sm text-gray-400">{file.size}</span>
                    </div>
                    <div className="w-full h-2 bg-gray-700/50 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-gradient-to-r from-blue-500 to-purple-600 rounded-full transition-all duration-300"
                        style={{ width: `${file.progress}%` }}
                      ></div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Files Section Header */}
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 mb-6">
            <h2 className="text-2xl font-bold text-white">My Files</h2>
            
            <div className="flex flex-wrap items-center gap-3">
              {/* Filter */}
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as FilterType)}
                className="px-4 py-2 rounded-xl bg-white/5 border border-white/10 text-white outline-none hover:bg-white/10 transition-all backdrop-blur-xl cursor-pointer"
              >
                <option value="all">All Files</option>
                <option value="image">Images</option>
                <option value="document">Documents</option>
                <option value="video">Videos</option>
                <option value="audio">Audio</option>
                <option value="archive">Archives</option>
              </select>

              {/* Sort */}
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as SortOption)}
                className="px-4 py-2 rounded-xl bg-white/5 border border-white/10 text-white outline-none hover:bg-white/10 transition-all backdrop-blur-xl cursor-pointer"
              >
                <option value="date">Sort by Date</option>
                <option value="name">Sort by Name</option>
                <option value="size">Sort by Size</option>
              </select>

              {/* View Mode Toggle */}
              <div className="flex gap-2 p-1 rounded-xl bg-white/5 border border-white/10">
                <button
                  onClick={() => setViewMode('grid')}
                  className={`p-2 rounded-lg transition-all ${
                    viewMode === 'grid'
                      ? 'bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  <Grid size={18} />
                </button>
                <button
                  onClick={() => setViewMode('list')}
                  className={`p-2 rounded-lg transition-all ${
                    viewMode === 'list'
                      ? 'bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  <List size={18} />
                </button>
              </div>
            </div>
          </div>

          {/* Files Grid/List - Continued in next file */}
          <div className={viewMode === 'grid' ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4' : 'space-y-3'}>
            {files.map((file) => {
              const FileIcon = getFileIcon(file.type);
              
              if (viewMode === 'grid') {
                return (
                  <div
                    key={file.id}
                    className="group relative p-5 rounded-2xl bg-white/5 backdrop-blur-xl border border-white/10 hover:bg-white/10 hover:border-blue-500/50 hover:shadow-xl hover:shadow-blue-500/20 transition-all duration-300 cursor-pointer hover:-translate-y-1"
                  >
                    {/* File Icon */}
                    <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                      <FileIcon className="text-blue-400" size={32} />
                    </div>

                    {/* File Info */}
                    <h3 className="text-white font-medium text-sm mb-2 truncate">{file.name}</h3>
                    <div className="flex items-center justify-between text-xs text-gray-400 mb-4">
                      <span>{file.size}</span>
                      <span>{file.date}</span>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-2">
                      <button className="flex-1 py-2 rounded-lg bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 text-xs font-medium transition-all flex items-center justify-center gap-1">
                        <Share2 size={14} />
                        Share
                      </button>
                      <button className="p-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white transition-all">
                        <MoreVertical size={16} />
                      </button>
                    </div>

                    {/* Favorite Badge */}
                    {file.favorite && (
                      <div className="absolute top-3 right-3">
                        <Star className="text-yellow-400 fill-yellow-400" size={16} />
                      </div>
                    )}
                  </div>
                );
              } else {
                return (
                  <div
                    key={file.id}
                    className="flex items-center gap-4 p-4 rounded-2xl bg-white/5 backdrop-blur-xl border border-white/10 hover:bg-white/10 hover:border-blue-500/50 transition-all cursor-pointer group"
                  >
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 flex items-center justify-center flex-shrink-0">
                      <FileIcon className="text-blue-400" size={24} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="text-white font-medium text-sm truncate">{file.name}</h3>
                      <p className="text-xs text-gray-400">{file.size} • {file.date}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      {file.favorite && <Star className="text-yellow-400 fill-yellow-400" size={16} />}
                      <button className="p-2 rounded-lg hover:bg-white/10 text-gray-400 hover:text-white transition-all">
                        <Download size={18} />
                      </button>
                      <button className="p-2 rounded-lg hover:bg-white/10 text-gray-400 hover:text-white transition-all">
                        <Share2 size={18} />
                      </button>
                      <button className="p-2 rounded-lg hover:bg-white/10 text-gray-400 hover:text-white transition-all">
                        <MoreVertical size={18} />
                      </button>
                    </div>
                  </div>
                );
              }
            })}
          </div>
        </main>
      </div>
    </div>
  );
};

export default FileShareDashboard;

