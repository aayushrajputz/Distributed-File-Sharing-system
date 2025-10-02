'use client'

import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  Shield, 
  Upload, 
  Share2, 
  Bell, 
  Zap, 
  Globe, 
  Lock, 
  Cloud,
  ArrowRight,
  CheckCircle,
  Star,
  Users,
  FileText,
  Smartphone,
  Laptop
} from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { useEffect, useState } from 'react'

export default function Home() {
  const { isAuthenticated } = useAuthStore()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) {
    return null
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-indigo-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      {/* Header */}
      <header className="border-b bg-card/50 backdrop-blur-sm">
        <div className="mx-auto max-w-7xl px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
                </svg>
              </div>
              <h1 className="text-xl font-bold">FileShare</h1>
            </div>

            <div className="flex items-center space-x-4">
              {isAuthenticated() ? (
                <div className="flex items-center space-x-2">
                  <Link href="/dashboard">
                    <Button>Go to Dashboard</Button>
                  </Link>
                </div>
              ) : (
                <div className="flex items-center space-x-2">
                  <Link href="/auth/login">
                    <Button variant="ghost">Sign In</Button>
                  </Link>
                  <Link href="/auth/register">
                    <Button>Get Started</Button>
                  </Link>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <main className="mx-auto max-w-7xl px-4 py-16">
        <div className="text-center mb-16">
          <Badge variant="secondary" className="mb-4 animate-fade-in">
            <Star className="w-3 h-3 mr-1" />
            Trusted by 10,000+ users
          </Badge>
          
          <h1 className="text-5xl md:text-6xl font-bold text-gray-900 dark:text-white mb-6 animate-slide-up">
            Share files
            <span className="bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent"> securely</span>
            <br />
            with anyone, anywhere
          </h1>
          
          <p className="text-xl text-gray-600 dark:text-gray-400 mb-8 max-w-3xl mx-auto animate-slide-up">
            A modern, production-grade file sharing platform built with microservices architecture. 
            Upload, share, and collaborate on files with enterprise-grade security and real-time notifications.
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center mb-12 animate-slide-up">
            {isAuthenticated() ? (
              <Link href="/dashboard">
                <Button size="lg" className="w-full sm:w-auto">
                  <Upload className="w-5 h-5 mr-2" />
                  Upload Files
                  <ArrowRight className="w-5 h-5 ml-2" />
                </Button>
              </Link>
            ) : (
              <>
                <Link href="/auth/register">
                  <Button size="lg" className="w-full sm:w-auto">
                    <Upload className="w-5 h-5 mr-2" />
                    Start Sharing
                    <ArrowRight className="w-5 h-5 ml-2" />
                  </Button>
                </Link>
                <Link href="/auth/login">
                  <Button size="lg" variant="outline" className="w-full sm:w-auto">
                    Sign In
                  </Button>
                </Link>
              </>
            )}
          </div>

          {/* Stats */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-2xl mx-auto animate-fade-in">
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600 mb-2">10K+</div>
              <div className="text-sm text-muted-foreground">Active Users</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600 mb-2">1M+</div>
              <div className="text-sm text-muted-foreground">Files Shared</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600 mb-2">99.9%</div>
              <div className="text-sm text-muted-foreground">Uptime</div>
            </div>
          </div>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mb-16">
          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Shield className="w-6 h-6 text-blue-600" />
              </div>
              <CardTitle>Enterprise Security</CardTitle>
              <CardDescription>
                JWT-based authentication with bcrypt password hashing and role-based access control
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  End-to-end encryption
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Multi-factor authentication
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Audit logs
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-green-100 dark:bg-green-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Upload className="w-6 h-6 text-green-600" />
              </div>
              <CardTitle>Drag & Drop Upload</CardTitle>
              <CardDescription>
                Intuitive file upload with progress tracking and batch processing
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Multiple file types
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Real-time progress
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Resume uploads
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Share2 className="w-6 h-6 text-purple-600" />
              </div>
              <CardTitle>Smart Sharing</CardTitle>
              <CardDescription>
                Share files with granular permissions and expiration dates
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Permission controls
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Expiration dates
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Password protection
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-orange-100 dark:bg-orange-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Bell className="w-6 h-6 text-orange-600" />
              </div>
              <CardTitle>Real-time Notifications</CardTitle>
              <CardDescription>
                Stay updated with instant notifications powered by Kafka
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Instant alerts
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Email notifications
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Activity tracking
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-indigo-100 dark:bg-indigo-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Zap className="w-6 h-6 text-indigo-600" />
              </div>
              <CardTitle>High Performance</CardTitle>
              <CardDescription>
                Built with microservices architecture for scalability and reliability
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Microservices
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Auto-scaling
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Load balancing
                </li>
              </ul>
            </CardContent>
          </Card>

          <Card className="card-modern hover:shadow-lg transition-all duration-200 group">
            <CardHeader>
              <div className="w-12 h-12 bg-pink-100 dark:bg-pink-900/20 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                <Globe className="w-6 h-6 text-pink-600" />
              </div>
              <CardTitle>Global Access</CardTitle>
              <CardDescription>
                Access your files from anywhere with our responsive web interface
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Mobile responsive
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Cross-platform
                </li>
                <li className="flex items-center">
                  <CheckCircle className="w-4 h-4 text-green-500 mr-2" />
                  Offline support
                </li>
              </ul>
            </CardContent>
          </Card>
        </div>

        {/* CTA Section */}
        <div className="text-center bg-gradient-to-r from-blue-600 to-indigo-600 rounded-2xl p-12 text-white">
          <h2 className="text-3xl font-bold mb-4">Ready to get started?</h2>
          <p className="text-xl mb-8 opacity-90">
            Join thousands of users who trust FileShare for their file sharing needs
          </p>
          {isAuthenticated() ? (
            <Link href="/dashboard">
              <Button size="lg" variant="secondary">
                <Upload className="w-5 h-5 mr-2" />
                Upload Your First File
              </Button>
            </Link>
          ) : (
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <Link href="/auth/register">
                <Button size="lg" variant="secondary">
                  Create Free Account
                </Button>
              </Link>
              <Link href="/auth/login">
                <Button size="lg" variant="outline" className="bg-transparent border-white text-white hover:bg-white hover:text-blue-600">
                  Sign In
                </Button>
              </Link>
            </div>
          )}
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-card/50 backdrop-blur-sm mt-16">
        <div className="mx-auto max-w-7xl px-4 py-8">
          <div className="text-center text-muted-foreground">
            <p>&copy; 2024 FileShare. Built with modern technologies for the future.</p>
          </div>
        </div>
      </footer>
    </div>
  )
}

