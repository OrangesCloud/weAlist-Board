# Database Migrations

## ⚠️ MIGRATION POLICY CHANGE

**As of November 2025, this project uses GORM auto-migration exclusively.**

All SQL migration files have been archived to the `archived/` directory. The database schema is now managed automatically by GORM based on the Go struct definitions in `internal/domain/`.

### Why the Change?

We experienced conflicts between SQL migrations and GORM auto-migration during deployments, causing "relation already exists" errors. To resolve this and simplify schema management, we've adopted a single-source-of-truth approach using GORM.

### Archived Migration Files

The following SQL migration files are preserved in `archived/` for historical reference:

- `001_init_schema.sql` - Initial schema creation (up migration)
- `001_init_schema_down.sql` - Schema rollback (down migration)
- `002_add_project_members_and_board_fields.sql` - Add project members, join requests, and board fields (up migration)
- `002_add_project_members_and_board_fields_down.sql` - Rollback project members and board fields (down migration)
- `003_migrate_existing_project_owners.sql` - Create OWNER members for existing projects (up migration)
- `003_migrate_existing_project_owners_down.sql` - Remove migrated OWNER members (down migration)
- `004_add_field_options.sql` - Add field options table (up migration)
- `004_add_field_options_down.sql` - Rollback field options (down migration)
- `005_add_custom_fields_to_boards.sql` - Add custom fields to boards (up migration)
- `005_add_custom_fields_to_boards_down.sql` - Rollback custom fields (down migration)
- `005_add_project_id_to_field_options.sql` - Add project_id to field options (up migration)
- `005_add_project_id_to_field_options_down.sql` - Rollback project_id addition (down migration)

## Schema Overview

The migrations create the following tables:

1. **projects** - Projects within workspaces (with owner_id and is_public fields)
2. **boards** - Work items with stage, importance, and role attributes (with author_id, assignee_id, and due_date fields)
3. **participants** - Users participating in boards
4. **comments** - Discussion comments on boards
5. **project_members** - Members of projects with roles (OWNER, ADMIN, MEMBER)
6. **project_join_requests** - Requests to join projects with approval workflow

## GORM Auto-Migration

### How It Works

When the board-service application starts, GORM automatically:

1. Checks if tables exist
2. Creates missing tables based on Go struct definitions
3. Adds missing columns to existing tables
4. Creates indexes and constraints

The migration logic is implemented in `internal/database/automigrate.go` with the `SafeAutoMigrate` function.

### Schema Source of Truth

The database schema is defined by Go structs in `internal/domain/`:

- `project.go` - Projects table
- `board.go` - Boards table
- `participant.go` - Participants table
- `comment.go` - Comments table
- `field_option.go` - Field options table

### Making Schema Changes

To modify the database schema:

1. Update the Go struct in `internal/domain/`
2. Add GORM tags for constraints, indexes, etc.
3. Restart the application - GORM will apply changes automatically

Example:
```go
type Project struct {
    BaseModel
    WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index"`
    Name        string    `gorm:"type:varchar(255);not null"`
    NewField    string    `gorm:"type:text"` // Add new field here
}
```

### Resetting the Database

If you need to completely reset the database (e.g., when switching from SQL migrations to GORM):

```bash
# Run the reset script
./scripts/reset-database.sh

# Or manually:
psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS project_board;"
psql -U postgres -d postgres -c "CREATE DATABASE project_board;"
```

After resetting, start the application and GORM will create all tables automatically.

## Features

### Automatic Timestamps

All tables include automatic `updated_at` timestamp updates via database triggers.

### Soft Deletes

All tables support soft deletes through the `deleted_at` column. Indexes are optimized for queries that filter out soft-deleted records.

### Constraints

- **Foreign Keys**: Cascade deletes to maintain referential integrity
- **Unique Constraints**: Prevent duplicate participants per board
- **Check Constraints**: Validate enum values for stage, importance, and role

### Indexes

Optimized indexes for:
- Foreign key lookups
- Soft delete filtering
- Workspace and project queries
- Board filtering by stage, importance, and role
- Comment ordering by creation time

## Schema Diagram

```
projects (1) ──< (N) boards (1) ──< (N) participants
    │                    │
    │                    └──< (N) comments
    │
    ├──< (N) project_members
    │
    └──< (N) project_join_requests
```

## Archived SQL Migrations

The `archived/` directory contains historical SQL migration files that were used before switching to GORM auto-migration. These files are kept for:

- Historical reference
- Understanding schema evolution
- Emergency rollback scenarios (if needed)

**Do not use these files for new deployments.** GORM auto-migration handles all schema management.

## Notes

- All IDs use UUID type with automatic generation
- The `pgcrypto` extension is required for UUID generation (GORM handles this)
- Timestamps use PostgreSQL's `TIMESTAMP` type (without timezone)
- All tables follow the soft delete pattern with `deleted_at` column
- GORM automatically creates indexes and constraints based on struct tags
