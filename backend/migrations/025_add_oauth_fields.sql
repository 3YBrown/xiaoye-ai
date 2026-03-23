-- Add OAuth provider ID columns for third-party login
ALTER TABLE users ADD COLUMN linuxdo_id VARCHAR(100) DEFAULT NULL;
ALTER TABLE users ADD UNIQUE INDEX idx_users_linuxdo_id (linuxdo_id);

-- Allow NULL password_hash for OAuth-only users
ALTER TABLE users MODIFY COLUMN password_hash VARCHAR(255) NULL;
