/**
 * Email validation utilities for real-time validation
 */

export interface EmailValidationResult {
  isValid: boolean;
  error?: string;
}

export interface EmailListValidationResult {
  isValid: boolean;
  validEmails: string[];
  invalidEmails: string[];
  errors: string[];
}

/**
 * Validates a single email address
 */
export function validateEmail(email: string): EmailValidationResult {
  if (!email || typeof email !== 'string') {
    return { isValid: false, error: 'Email is required' };
  }

  const trimmedEmail = email.trim();
  
  if (trimmedEmail.length === 0) {
    return { isValid: false, error: 'Email cannot be empty' };
  }

  // Basic email regex pattern
  const emailRegex = /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
  
  if (!emailRegex.test(trimmedEmail)) {
    return { isValid: false, error: 'Invalid email format' };
  }

  // Check length limits
  if (trimmedEmail.length > 254) {
    return { isValid: false, error: 'Email is too long' };
  }

  const [localPart, domain] = trimmedEmail.split('@');
  if (localPart.length > 64) {
    return { isValid: false, error: 'Email local part is too long' };
  }

  return { isValid: true };
}

/**
 * Validates a comma-separated list of email addresses
 */
export function validateEmailList(emailList: string): EmailListValidationResult {
  if (!emailList || typeof emailList !== 'string') {
    return {
      isValid: false,
      validEmails: [],
      invalidEmails: [],
      errors: ['Email list is required']
    };
  }

  const trimmedList = emailList.trim();
  
  // Allow empty string for "share link only" option
  if (trimmedList.length === 0) {
    return {
      isValid: true,
      validEmails: [],
      invalidEmails: [],
      errors: []
    };
  }

  const emails = trimmedList
    .split(',')
    .map(email => email.trim())
    .filter(email => email.length > 0);

  const validEmails: string[] = [];
  const invalidEmails: string[] = [];
  const errors: string[] = [];

  for (const email of emails) {
    const validation = validateEmail(email);
    if (validation.isValid) {
      validEmails.push(email);
    } else {
      invalidEmails.push(email);
      errors.push(`${email}: ${validation.error}`);
    }
  }

  return {
    isValid: invalidEmails.length === 0,
    validEmails,
    invalidEmails,
    errors
  };
}

/**
 * Formats email list for display (trims and removes duplicates)
 */
export function formatEmailList(emailList: string): string {
  if (!emailList || typeof emailList !== 'string') {
    return '';
  }

  const emails = emailList
    .split(',')
    .map(email => email.trim())
    .filter(email => email.length > 0);

  // Remove duplicates
  const uniqueEmails = [...new Set(emails)];

  return uniqueEmails.join(', ');
}

/**
 * Checks if email list is empty (for "share link only" option)
 */
export function isEmailListEmpty(emailList: string): boolean {
  if (!emailList || typeof emailList !== 'string') {
    return true;
  }

  const trimmed = emailList.trim();
  return trimmed.length === 0;
}




