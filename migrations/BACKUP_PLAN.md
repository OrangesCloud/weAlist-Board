# Database Migration Backup Plan

## Overview
This document outlines the backup and rollback strategy for Migration 005, which removes legacy columns (stage, importance, role) from the boards table.

## Pre-Migration Checklist

### 1. Database Backup
Before applying migration 005 to production:

```bash
# Create a full database backup
pg_dump -h <host> -U <user> -d <database> -F c -f backup_before_migration_005_$(date +%Y%m%d_%H%M%S).dump

# Verify backup file was created
ls -lh backup_before_migration_005_*.dump
```

### 2. Verify Data Integrity
Ensure all data has been migrated to custom_fields:

```sql
-- Check if any boards have NULL custom_fields
SELECT COUNT(*) FROM boards WHERE custom_fields IS NULL OR custom_fields = '{}'::jsonb;

-- Verify custom_fields contains expected keys
SELECT 
    COUNT(*) as total_boards,
    COUNT(CASE WHEN custom_fields ? 'stage' THEN 1 END) as has_stage,
    COUNT(CASE WHEN custom_fields ? 'importance' THEN 1 END) as has_importance,
    COUNT(CASE WHEN custom_fields ? 'role' THEN 1 END) as has_role
FROM boards;

-- Sample data comparison (if legacy columns still exist)
SELECT 
    id,
    stage as old_stage,
    custom_fields->>'stage' as new_stage,
    importance as old_importance,
    custom_fields->>'importance' as new_importance,
    role as old_role,
    custom_fields->>'role' as new_role
FROM boards
LIMIT 10;
```

### 3. Test in Staging Environment
1. Apply migration to staging database
2. Run full test suite
3. Verify API endpoints work correctly
4. Test rollback procedure
5. Re-apply migration after successful rollback test

## Migration Execution

### Development Environment
```bash
cd board-service
# Apply migration
psql -h localhost -U postgres -d board_service_dev -f migrations/005_add_custom_fields_to_boards.sql
```

### Production Environment
```bash
# 1. Create backup (see above)
# 2. Apply migration during maintenance window
psql -h <prod-host> -U <prod-user> -d <prod-database> -f migrations/005_add_custom_fields_to_boards.sql

# 3. Verify migration success
psql -h <prod-host> -U <prod-user> -d <prod-database> -c "\d boards"
```

## Rollback Procedure

### If Issues Detected Immediately After Migration

```bash
# Apply rollback script
psql -h <host> -U <user> -d <database> -f migrations/005_add_custom_fields_to_boards_down.sql

# Verify rollback success
psql -h <host> -U <user> -d <database> -c "\d boards"
```

### If Issues Detected After Extended Period

If the rollback script cannot be used (e.g., custom_fields column was modified):

```bash
# Restore from backup
pg_restore -h <host> -U <user> -d <database> -c backup_before_migration_005_<timestamp>.dump

# Or restore to a new database and migrate data
createdb <database>_restored
pg_restore -h <host> -U <user> -d <database>_restored backup_before_migration_005_<timestamp>.dump
```

## Post-Migration Verification

### 1. Check Table Structure
```sql
-- Verify legacy columns are removed
\d boards

-- Expected: custom_fields column exists, stage/importance/role columns do not exist
```

### 2. Verify Data Integrity
```sql
-- Check all boards have custom_fields
SELECT COUNT(*) FROM boards WHERE custom_fields IS NULL;
-- Expected: 0

-- Verify custom_fields structure
SELECT 
    custom_fields->>'stage' as stage,
    custom_fields->>'importance' as importance,
    custom_fields->>'role' as role,
    COUNT(*) as count
FROM boards
GROUP BY custom_fields->>'stage', custom_fields->>'importance', custom_fields->>'role'
ORDER BY count DESC;
```

### 3. API Testing
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

### 4. Performance Testing
```sql
-- Test JSONB query performance
EXPLAIN ANALYZE
SELECT * FROM boards 
WHERE custom_fields->>'stage' = 'in_progress'
AND custom_fields->>'importance' = 'urgent';

-- Verify GIN index is being used
-- Expected: "Bitmap Index Scan on idx_boards_custom_fields"
```

## Monitoring

### Key Metrics to Monitor Post-Migration
1. API response times for board queries
2. Database query performance
3. Error rates in application logs
4. Custom fields data consistency

### Alert Conditions
- Increase in 500 errors from board service
- Slow query alerts for boards table
- NULL custom_fields detected
- Failed board creation/update operations

## Contact Information

### Escalation Path
1. **Development Team**: Check application logs and API errors
2. **Database Team**: Verify migration status and data integrity
3. **DevOps Team**: Coordinate rollback if necessary

## Lessons Learned (Post-Migration)

Document any issues encountered and resolutions:
- [ ] Migration execution time: _____ minutes
- [ ] Downtime duration: _____ minutes
- [ ] Issues encountered: _____
- [ ] Resolution steps: _____
- [ ] Improvements for future migrations: _____
