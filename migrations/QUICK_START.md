# Quick Start: Database Migration for Task 18

## TL;DR

✅ **No SQL files needed** - This project uses GORM auto-migration

✅ **Schema changes already in code** - Domain models updated in previous tasks

✅ **Migration happens automatically** - When you start the application

## What You Need to Know

### 1. Schema Changes Are Already Defined

The following domain models have been updated:

- `internal/domain/board.go` → Added `StartDate` field
- `internal/domain/project.go` → Added `StartDate` and `DueDate` fields  
- `internal/domain/attachment.go` → Complete new table

### 2. How to Apply the Changes

Simply start the application:

```bash
cd board-service
make run
```

GORM will automatically:
- Add `start_date` column to `boards` table
- Add `start_date` and `due_date` columns to `projects` table
- Create `attachments` table with all columns and indexes

### 3. How to Verify

**Option A: Check application logs**
```
INFO  Database schema migration completed successfully
```

**Option B: Run verification script**
```bash
./scripts/verify-schema-changes.sh
```

**Option C: Check database directly**
```bash
psql -U postgres -d project_board
\d boards
\d projects
\d attachments
```

## Why No SQL Migration Files?

This project switched to GORM auto-migration in November 2025 to avoid conflicts between SQL migrations and GORM. See `migrations/README.md` for details.

## Need More Info?

- **Detailed explanation**: See `GORM_AUTO_MIGRATION_NOTES.md`
- **Complete summary**: See `TASK_18_COMPLETION_SUMMARY.md`
- **Migration policy**: See `README.md`

## Troubleshooting

**Problem**: Migration fails on startup

**Solution**: Check database connection and permissions

```bash
# Test database connection
psql -U postgres -d project_board -c "SELECT 1;"

# Check application logs for detailed error
make run
```

**Problem**: Columns not appearing

**Solution**: Ensure you're using the latest code and restart the application

```bash
git pull
make run
```

## That's It!

The migration is complete. Just start the application and GORM handles the rest.
