# Configuration Files

This directory contains configuration files for the Project Board Management API.

## Files

### config.yaml
The main configuration file used by the application. This file is loaded at startup and contains all application settings.

**Note**: This file is not tracked in git. Create it from the example file.

### config.yaml.example
Example configuration file with all available options and documentation. Use this as a template to create your `config.yaml`.

**Setup**:
```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your settings
```

## Configuration Methods

The application supports two configuration methods:

1. **YAML Configuration File** (`config.yaml`)
   - Structured configuration
   - Easy to read and maintain
   - Good for complex settings

2. **Environment Variables** (`.env` or system environment)
   - Override YAML settings
   - Good for secrets and deployment-specific values
   - Required for Docker/Kubernetes deployments

## Priority

Configuration is loaded in this order (later sources override earlier ones):

1. Default values in code
2. `config.yaml` file
3. Environment variables

## Quick Start

### Development

```bash
# Copy example config
cp config.yaml.example config.yaml

# Edit with your local settings
# Default values work for local PostgreSQL
```

### Production

```bash
# Use config.yaml for base settings
cp config.yaml.example config.yaml

# Override secrets with environment variables
export DB_PASSWORD="secure-password"
export JWT_SECRET="secure-random-secret"

# Run application
./main
```

### Docker

```bash
# Use environment variables only
docker run -e DB_HOST=postgres \
           -e DB_PASSWORD=secure \
           -e JWT_SECRET=secret \
           project-board-api
```

## Documentation

For detailed configuration documentation, see:
- [Configuration Guide](../docs/CONFIGURATION.md) - Complete reference
- [.env.example](../.env.example) - Environment variables template
- [README.md](../README.md) - Quick start guide

## Security

⚠️ **Important Security Notes**:

1. **Never commit secrets** to version control
2. `config.yaml` is in `.gitignore` for this reason
3. Use environment variables for sensitive data:
   - `DB_PASSWORD`
   - `JWT_SECRET`
4. Generate strong JWT secrets: `openssl rand -base64 32`
5. Use different secrets for each environment

## Validation

The application validates configuration on startup. Required fields:

- `server.port`
- `database.host`
- `database.port`
- `database.user`
- `database.dbname`
- `jwt.secret`

If validation fails, the application will exit with a clear error message.

## Examples

### Minimal config.yaml

```yaml
server:
  port: "8080"
  mode: "debug"

database:
  host: "localhost"
  port: "5432"
  user: "postgres"
  password: "password"
  dbname: "project_board"

logger:
  level: "info"

jwt:
  secret: "dev-secret-change-in-production"
```

### Production config.yaml (with env vars for secrets)

```yaml
server:
  port: "8080"
  mode: "release"
  read_timeout: 15s
  write_timeout: 15s
  shutdown_timeout: 30s

database:
  host: "postgres.internal"
  port: "5432"
  user: "app_user"
  # password: set via DB_PASSWORD env var
  dbname: "project_board"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 10m

logger:
  level: "info"
  output_path: "/var/log/app.log"

jwt:
  # secret: set via JWT_SECRET env var
  expire_time: 8h
```

Then set environment variables:
```bash
export DB_PASSWORD="secure-database-password"
export JWT_SECRET="secure-jwt-secret-key"
```

## Troubleshooting

### "failed to read config file"
- Ensure `config.yaml` exists in the `configs/` directory
- Check file permissions

### "invalid configuration: database host is required"
- Ensure all required fields are set
- Check for typos in field names

### "failed to parse config file"
- Validate YAML syntax
- Check indentation (use spaces, not tabs)
- Ensure proper quoting of string values

## Support

For more help:
- See [Configuration Guide](../docs/CONFIGURATION.md)
- Check application logs for detailed errors
- Review [README.md](../README.md) for setup instructions
