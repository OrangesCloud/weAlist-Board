# Attachment Model Migration Summary

## Overview
This document describes the changes made to the Attachment model to support temporary file management and entity linking.

## Changes Made

### 1. Domain Model Updates (`internal/domain/attachment.go`)

#### New Type: AttachmentStatus
```go
type AttachmentStatus string

const (
    AttachmentStatusTemp      AttachmentStatus = "TEMP"      // Temporary status
    AttachmentStatusConfirmed AttachmentStatus = "CONFIRMED" // Confirmed status
)
```

#### Updated Attachment Struct
The following fields were modified or added:

1. **EntityID** - Changed from `uuid.UUID` to `*uuid.UUID` (nullable)
   - Allows attachments to be created without being linked to an entity
   - Will be set when the entity (board/comment/project) is created

2. **Status** - Added `AttachmentStatus` field
   - Type: `varchar(20)`
   - Default: `'TEMP'`
   - Index: `idx_attachments_status`
   - Tracks whether attachment is temporary or confirmed

3. **ExpiresAt** - Added `*time.Time` field
   - Type: `timestamp`
   - Nullable
   - Index: `idx_attachments_expires_at`
   - Set to 1 hour after creation for temporary attachments

### 2. Handler Updates (`internal/handler/attachment_handler.go`)

Updated `SaveAttachmentMetadata` function to:
- Set `EntityID` to `nil` for temporary attachments
- Set `Status` to `AttachmentStatusTemp`
- Calculate and set `ExpiresAt` to 1 hour from creation

### 3. Test Updates

#### Integration Test Schema (`internal/handler/integration_test.go`)
Updated the test database schema to include:
- `entity_id TEXT` (nullable, removed NOT NULL constraint)
- `status TEXT NOT NULL DEFAULT 'TEMP'`
- `expires_at DATETIME` (nullable)

#### Test Data
Updated test attachments to use:
- Pointer type for `EntityID` (e.g., `&board.ID`)
- `Status` field set to `AttachmentStatusConfirmed`
- `ExpiresAt` set to `nil` for confirmed attachments

## Migration Behavior

### GORM Auto-Migration
Since this project uses GORM auto-migration (not SQL migration files), the schema changes will be applied automatically when the application starts:

1. **Existing Tables**: GORM will add the new columns (`status`, `expires_at`) and modify `entity_id` to be nullable
2. **New Tables**: GORM will create the table with all fields including the new ones

### Database Changes
When the application starts, GORM will execute:
```sql
-- Make entity_id nullable
ALTER TABLE attachments ALTER COLUMN entity_id DROP NOT NULL;

-- Add status column with default value
ALTER TABLE attachments ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'TEMP';

-- Add expires_at column
ALTER TABLE attachments ADD COLUMN expires_at TIMESTAMP;

-- Add indexes
CREATE INDEX idx_attachments_status ON attachments(status);
CREATE INDEX idx_attachments_expires_at ON attachments(expires_at);
```

## Workflow

### Temporary Attachment Flow
1. User requests presigned URL
2. User uploads file to S3
3. Backend saves attachment metadata with:
   - `EntityID = nil`
   - `Status = TEMP`
   - `ExpiresAt = now + 1 hour`
4. User creates board/comment/project with attachment IDs
5. Backend updates attachments:
   - `EntityID = <entity_id>`
   - `Status = CONFIRMED`
   - `ExpiresAt = nil`

### Cleanup Job
A background job will run hourly to:
1. Find attachments where `Status = TEMP` AND `ExpiresAt < NOW()`
2. Delete files from S3
3. Delete records from database

## Validation

### Tests Passing
- ✅ Integration test: `TestIntegration_AttachmentsRetrieval`
- ✅ Handler tests: All attachment handler tests pass
- ✅ Repository tests: All repository tests pass
- ✅ Code compilation: All packages compile successfully

### Verified Functionality
- ✅ Nullable EntityID works correctly
- ✅ Status field with default value works
- ✅ ExpiresAt field stores timestamps correctly
- ✅ Indexes are created properly

## Requirements Validated
This migration satisfies **Requirement 7.1** from the design document:
> WHEN a file is uploaded to S3 THEN the system SHALL mark the attachment as temporary with an expiration time of 1 hour

## Next Steps
The following tasks will build on this migration:
- Task 21: Implement repository methods for finding expired attachments
- Task 22: Update board/comment/project creation to link attachments
- Task 26: Implement cleanup job to delete expired temporary files
