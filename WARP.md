# Warp AI Rules for buildkite-forgejo-webhook

## Project Overview
A lightweight webhook bridge service that translates Forgejo/Gitea webhooks into Buildkite API calls.

## Technology Stack
- **Language**: Go 1.25
- **Dependencies**: 
  - `github.com/joho/godotenv` - Environment variable management
- **Deployment**: Docker Compose

## Development Guidelines

### Environment Configuration
- All configuration is managed through `.env` file
- The application automatically loads `.env` on startup using `godotenv`
- Never commit `.env` files - use `.env.example` for templates
- Environment variables can still be set directly (they take precedence over `.env`)

### Code Style
- Use emoji in log messages for better readability (🚀, 📡, 📨, ✅, ❌)
- Keep the codebase simple and stateless
- Use descriptive variable names
- Add comments for complex logic

### Testing
- Manual testing using curl commands (see README)
- Test webhook payloads should match Forgejo/Gitea format
- Always test with the `/health` endpoint after changes

### Docker
- Use `compose.yml` (modern naming convention, not `docker-compose.yml`)
- Service should expose port 8080 by default
- Include health checks in Docker configuration

### Documentation
- Keep README.md comprehensive with examples
- Document all environment variables
- Include troubleshooting section for common issues
- Use clear emoji for visual organization in docs

## Security Notes
- API tokens should never be committed
- Recommend running behind reverse proxy with SSL/TLS
- Document security considerations in README
