# GORM Auto-Migration Notes

## Task 18: Database Schema Changes

This document describes the database schema changes implemented for task 18 of the board-api-improvements feature.

### Overview

This project uses **GORM auto-migration** exclusively. Database schema changes are made by updating Go struct definitions in `internal/domain/`, and GORM automatically applies these changes when the application starts.

### Schema Changes Implemented

#### 1. Boards Table - Added `start_date` Column

**File**: `internal/domain/board.go`

```go
type Board struct {
    // ... existing fields ...
    StartDate    *time.Time             `gorm:"type:timestamp;index:idx_boards_start_date" json:"start_date"`
    DueDate      *time.Time             `gorm:"type:timestamp;index:idx_boards_due_date" json:"due_date"`
    // ... other fields ...
}
```

**Database Impact**:
- Column: `start_date` (nullable timestamp)
- Index: `idx_boards_start_date` for efficient date-based queries
- Allows boards to have an optional start date in addition to the existing due date

**Requirements**: 6.3

#### 2. Projects Table - Added `start_date` and `due_date` Columns

**File**: `internal/domain/project.go`

```go
type Project struct {
    // ... existing fields ...
    StartDate   *time.Time       `gorm:"type:timestamp" json:"start_date,omitempty"`
    DueDate     *time.Time       `gorm:"type:timestamp" json:"due_date,omitempty"`
    // ... other fields ...
}
```

**Database Impact**:
- Column: `start_date` (nullable timestamp)
- Column: `due_date` (nullable timestamp)
- Allows projects to have optional start and due dates for timeline tracking

**Requirements**: 7.1, 7.2

#### 3. Attachments Table - New Table Created

**File**: `internal/domain/attachment.go`

```go
type Attachment struct {
    BaseModel
    EntityType  EntityType `gorm:"type:varchar(50);not null;index:idx_attachments_entity,priority:1"`
    EntityID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_attachments_entity,priority:2;index:idx_attachments_entity_id"`
    FileName    string     `gorm:"type:varchar(255);not null"`
    FileURL     string     `gorm:"type:text;not null"`
    FileSize    int64      `gorm:"not null"`
    ContentType string     `gorm:"type:varchar(100);not null"`
    UploadedBy  uuid.UUID  `gorm:"type:uuid;not null;index:idx_attachments_uploaded_by"`
}
```

**Database Impact**:
- New table: `attachments`
- Columns:
  - `id` (UUID, primary key, auto-generated)
  - `entity_type` (varchar(50), not null) - "BOARD" or "PROJECT"
  - `entity_id` (UUID, not null) - references board or project
  - `file_name` (varchar(255), not null) - original file name
  - `file_url` (text, not null) - S3 URL or storage path
  - `file_size` (bigint, not null) - file size in bytes
  - `content_type` (varchar(100), not null) - MIME type
  - `uploaded_by` (UUID, not null) - user who uploaded the file
  - `created_at` (timestamp, auto-managed by BaseModel)
  - `updated_at` (timestamp, auto-managed by BaseModel)
  - `deleted_at` (timestamp, nullable, for soft deletes)
- Indexes:
  - Composite index on `(entity_type, entity_id)` for efficient lookups
  - Index on `entity_id` for foreign key performance
  - Index on `uploaded_by` for user-based queries

**Requirements**: 8.4, 8.5

### Auto-Migration Configuration

All models are registered in `internal/database/automigrate.go`:

```go
models := []modelInfo{
    {&domain.Project{}, "projects"},
    {&domain.Board{}, "boards"},
    {&domain.Attachment{}, "attachments"},
    // ... other models ...
}
```

### How Auto-Migration Works

1. **Application Startup**: When the board-service starts, it calls `SafeAutoMigrateWithRetry()`
2. **Schema Detection**: GORM checks each table and compares it with the Go struct definition
3. **Automatic Updates**:
   - **New tables**: Created from scratch with all columns, indexes, and constraints
   - **Existing tables**: Only missing columns and indexes are added
   - **Existing columns**: Not modified (GORM doesn't alter or drop existing columns)
4. **Safe Operation**: The migration is idempotent - running it multiple times is safe

### Migration Execution

The migration runs automatically on every application startup:

```go
// From cmd/api/main.go
if err := database.SafeAutoMigrateWithRetry(db, log.Logger, 3); err != nil {
    log.Fatal("Failed to run auto-migration", zap.Error(err))
}
```

**Retry Logic**:
- Maximum 3 attempts
- Exponential backoff between retries
- Detailed logging for each attempt

### Verification

To verify the schema changes were applied:

```sql
-- Check boards table for start_date column
\d boards

-- Check projects table for start_date and due_date columns
\d projects

-- Check if attachments table exists
\d attachments

-- Verify indexes
\di idx_boards_start_date
\di idx_attachments_entity
\di idx_attachments_uploaded_by
```

### Rollback Considerations

Since GORM auto-migration only adds columns and tables (never removes them), rollback requires manual intervention:

```sql
-- To rollback boards.start_date (if needed)
ALTER TABLE boards DROP COLUMN IF EXISTS start_date;

-- To rollback projects date columns (if needed)
ALTER TABLE projects DROP COLUMN IF EXISTS start_date;
ALTER TABLE projects DROP COLUMN IF EXISTS due_date;

-- To rollback attachments table (if needed)
DROP TABLE IF EXISTS attachments CASCADE;
```

**Note**: Rollback should only be necessary in exceptional circumstances, as these are additive changes.

### Testing the Migration

1. **Start the application**:
   ```bash
   cd board-service
   make run
   ```

2. **Check logs** for migration success:
   ```
   INFO  Starting auto-migration with retry logic
   INFO  Table exists, updating schema only  table=boards
   INFO  Table exists, updating schema only  table=projects
   INFO  Table does not exist, creating new table  table=attachments
   INFO  Successfully migrated table  table=attachments
   INFO  Database schema migration completed successfully
   ```

3. **Verify in database**:
   ```bash
   psql -U postgres -d project_board
   \d boards
   \d projects
   \d attachments
   ```

### Related Files

- Domain models: `internal/domain/board.go`, `internal/domain/project.go`, `internal/domain/attachment.go`
- Auto-migration: `internal/database/automigrate.go`
- Application startup: `cmd/api/main.go`
- DTOs: `internal/dto/board_dto.go`, `internal/dto/project_dto.go`

### References

- Requirements: `.kiro/specs/board-api-improvements/requirements.md`
- Design: `.kiro/specs/board-api-improvements/design.md`
- Tasks: `.kiro/specs/board-api-improvements/tasks.md`
