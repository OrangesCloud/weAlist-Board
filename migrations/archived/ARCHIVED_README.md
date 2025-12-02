# Archived SQL Migrations

## Why These Files Are Archived

These SQL migration files were used in earlier versions of the board-service but have been **archived and are no longer active** as of November 2025.

## Reason for Archival

We experienced deployment issues caused by conflicts between SQL migrations and GORM auto-migration:

- **Problem**: SQL migrations created tables, then GORM tried to create them again
- **Error**: `ERROR: relation "projects" already exists`
- **Impact**: Service failed to start during deployments

## Current Migration Strategy

**The board-service now uses GORM auto-migration exclusively.**

- Schema is defined in Go structs (`internal/domain/`)
- GORM automatically creates/updates tables on application startup
- No manual SQL migration files needed
- Single source of truth for schema

## What's in This Directory

Historical SQL migration files:

1. **001_init_schema.sql** - Initial database schema
2. **002_add_project_members_and_board_fields.sql** - Project members and board enhancements
3. **003_migrate_existing_project_owners.sql** - Data migration for project owners
4. **004_add_field_options.sql** - Field options table
5. **005_add_custom_fields_to_boards.sql** - Custom fields for boards
6. **005_add_project_id_to_field_options.sql** - Project ID in field options

Each migration has a corresponding `_down.sql` file for rollback.

## Should You Use These Files?

**No.** These files are for historical reference only.

For new deployments:
1. Use the database reset script: `./scripts/reset-database.sh`
2. Start the board-service application
3. GORM will create all tables automatically

## Emergency Rollback

If you need to manually recreate the schema (emergency only):

```bash
# Reset database
psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS project_board;"
psql -U postgres -d postgres -c "CREATE DATABASE project_board;"

# Let GORM handle the rest - just start the application
```

## Questions?

See the main migrations README: `../README.md`

---

**Last Updated**: November 2025  
**Status**: Archived - Do Not Use
