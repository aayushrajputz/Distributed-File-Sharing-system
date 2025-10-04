-- Private Folder PIN Protection Database Schema
-- This schema extends the existing file sharing platform

-- Table for storing user PINs with security features
CREATE TABLE user_pins (
    id VARCHAR(24) PRIMARY KEY,
    user_id VARCHAR(24) NOT NULL,
    pin_hash VARCHAR(255) NOT NULL, -- bcrypt hashed PIN
    salt VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Brute force protection
    failed_attempts INT DEFAULT 0,
    locked_until TIMESTAMP NULL,
    
    -- PIN policy
    pin_length INT DEFAULT 4,
    expires_at TIMESTAMP NULL,
    
    INDEX idx_user_id (user_id),
    INDEX idx_active (is_active),
    INDEX idx_locked_until (locked_until)
);

-- Table for private folder access logs
CREATE TABLE private_folder_access_logs (
    id VARCHAR(24) PRIMARY KEY,
    user_id VARCHAR(24) NOT NULL,
    file_id VARCHAR(24) NOT NULL,
    action ENUM('PIN_VERIFIED', 'PIN_FAILED', 'FOLDER_ACCESSED', 'FILE_MOVED_TO_PRIVATE', 'FILE_MOVED_FROM_PRIVATE') NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(255) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_file_id (file_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at),
    INDEX idx_success (success)
);

-- Table for private folder file mappings
CREATE TABLE private_folder_files (
    id VARCHAR(24) PRIMARY KEY,
    user_id VARCHAR(24) NOT NULL,
    file_id VARCHAR(24) NOT NULL,
    original_folder_id VARCHAR(24) NULL, -- Store original location for restoration
    moved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_private BOOLEAN DEFAULT TRUE,
    
    UNIQUE KEY unique_user_file (user_id, file_id),
    INDEX idx_user_id (user_id),
    INDEX idx_file_id (file_id),
    INDEX idx_is_private (is_private)
);

-- Table for PIN attempt tracking (for brute force prevention)
CREATE TABLE pin_attempts (
    id VARCHAR(24) PRIMARY KEY,
    user_id VARCHAR(24) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    attempt_count INT DEFAULT 1,
    first_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_blocked BOOLEAN DEFAULT FALSE,
    blocked_until TIMESTAMP NULL,
    
    UNIQUE KEY unique_user_ip (user_id, ip_address),
    INDEX idx_user_id (user_id),
    INDEX idx_ip_address (ip_address),
    INDEX idx_blocked_until (blocked_until)
);

-- Update existing files table to support private folder
ALTER TABLE files ADD COLUMN is_private BOOLEAN DEFAULT FALSE;
ALTER TABLE files ADD COLUMN private_folder_id VARCHAR(24) NULL;
ALTER TABLE files ADD INDEX idx_is_private (is_private);
ALTER TABLE files ADD INDEX idx_private_folder_id (private_folder_id);

