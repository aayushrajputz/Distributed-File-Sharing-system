import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

export function formatDate(dateString: string): string {
  if (!dateString) return 'Unknown Date'
  
  const date = new Date(dateString)
  
  // Check if the date is valid
  if (isNaN(date.getTime())) {
    return 'Unknown Date'
  }
  
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function safeDateParse(dateString: string): Date {
  if (!dateString) return new Date(0) // Return epoch for invalid dates
  
  const date = new Date(dateString)
  
  // Check if the date is valid
  if (isNaN(date.getTime())) {
    return new Date(0) // Return epoch for invalid dates
  }
  
  return date
}

