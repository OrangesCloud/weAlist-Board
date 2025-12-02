# Migration 005 Verification Checklist

## Pre-Migration Verification

### ✅ Code Review
- [x] No references to `stage`, `importance`, or `role` as separate columns in Go code
- [x] All code uses `CustomFields` map instead
- [x] Domain model (board.go) only has `CustomFields` field
- [x] DTOs use `CustomFields` structure
- [x] Services handle `CustomFields` correctly
- [x] Handlers parse and return `CustomFields`

### ✅ Build Verification
- [x] Go build completes successfully: `go build ./...`
- [x] No compilation errors

### ✅ Test Verification
- [x] All board-related tests pass
- [x] All field option tests pass
- [x] All handler tests pass
- [x] CustomFields filtering tests pass
- [x] Error response consistency tests pass

**Note**: One pre-existing test failure in `project_member_service_test.go` (self-removal check) is unrelated to this migration.

## Migration File Verification

### ✅ Migration UP Script (005_add_custom_fields_to_boards.sql)
- [x] Step 1: Adds `custom_fields` JSONB column
- [x] Step 2: Migrates data from legacy columns to `custom_fields`
- [x] Step 3: Creates GIN index on `custom_fields`
- [x] Step 4: Drops old indexes (idx_boards_stage, idx_boards_importance, idx_boards_role)
- [x] Step 5: Drops legacy columns (stage, importance, role)
- [x] Includes proper comments

### ✅ Migration DOWN Script (005_add_custom_fields_to_boards_down.sql)
- [x] Step 1: Recreates legacy columns
- [x] Step 2: Restores data from `custom_fields` to legacy columns
- [x] Step 3: Adds NOT NULL constraints with default values
- [x] Step 4: Adds CHECK constraints for valid values
- [x] Step 5: Recreates indexes on legacy columns
- [x] Step 6: Drops GIN index on `custom_fields`
- [x] Step 7: Drops `custom_fields` column

## Documentation Verification

### ✅ Supporting Documents
- [x] BACKUP_PLAN.md created with comprehensive backup strategy
- [x] Pre-migration checklist included
- [x] Rollback procedures documented
- [x] Post-migration verification steps included
- [x] SQL verification queries provided
- [x] API testing examples included

## Post-Migration Verification Steps

### Database Structure
```sql
-- Verify legacy columns are removed
\d boards

-- Expected output should NOT include:
-- - stage column
-- - importance column  
-- - role column

-- Expected output SHOULD include:
-- - custom_fields jsonb column
```

### Data Integrity
```sql
-- Check all boards have custom_fields populated
SELECT COUNT(*) FROM boards WHERE custom_fields IS NULL OR custom_fields = '{}'::jsonb;
-- Expected: 0

-- Verify custom_fields structure
SELECT 
    id,
    custom_fields->>'stage' as stage,
    custom_fields->>'importance' as importance,
    custom_fields->>'role' as role
FROM boards
LIMIT 5;
-- Expected: All rows should have values in custom_fields
```

### Index Verification
```sql
-- Check GIN index exists
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'boards' AND indexname = 'idx_boards_custom_fields';
-- Expected: 1 row with GIN index definition

-- Check old indexes are removed
SELECT indexname 
FROM pg_indexes 
WHERE tablename = 'boards' 
AND indexname IN ('idx_boards_stage', 'idx_boards_importance', 'idx_boards_role');
-- Expected: 0 rows
```

### API Testing
```bash
# Test board creation with custom fields
curl -X POST http://localhost:8080/api/boards \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "projectId": "<uuid>",
    "title": "Test Board",
    "customFields": {
      "stage": "in_progress",
      "importance": "urgent",
      "role": "developer"
    }
  }'

# Test board filtering by custom fields
curl "http://localhost:8080/api/boards?projectId=<uuid>&customFields=%7B%22stage%22%3A%22in_progress%22%7D" \
  -H "Authorization: Bearer <token>"
```

## Rollback Testing

### Test Rollback Procedure
```bash
# Apply rollback
psql -h localhost -U postgres -d board_service_dev \
  -f migrations/005_add_custom_fields_to_boards_down.sql

# Verify legacy columns restored
psql -h localhost -U postgres -d board_service_dev -c "\d boards"

# Verify data restored
psql -h localhost -U postgres -d board_service_dev -c \
  "SELECT id, stage, importance, role FROM boards LIMIT 5;"

# Re-apply migration
psql -h localhost -U postgres -d board_service_dev \
  -f migrations/005_add_custom_fields_to_boards.sql
```

## Sign-off

- [x] All code changes reviewed and approved
- [x] All tests passing (except pre-existing failure)
- [x] Migration scripts verified
- [x] Rollback scripts verified
- [x] Documentation complete
- [x] Backup plan established

**Status**: ✅ Ready for deployment

**Notes**:
- One pre-existing test failure in `TestProjectMemberService_RemoveMember/실패:_자기_자신_제거_시도` is unrelated to this migration
- All custom fields functionality tests pass
- Build completes successfully
- No references to legacy columns remain in codebase
