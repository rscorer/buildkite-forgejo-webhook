# Buildkite-Forgejo Webhook Bridge

A lightweight webhook bridge service that enables [Forgejo](https://forgejo.org/) and [Gitea](https://about.gitea.com/) repositories to trigger [Buildkite](https://buildkite.com/) builds automatically on push events.

## 🎯 Problem

Buildkite doesn't natively support Forgejo/Gitea webhooks - it primarily integrates with GitHub, GitLab, and Bitbucket through their apps. If you're running Forgejo or Gitea for your self-hosted git infrastructure, you need a way to trigger Buildkite builds on push events.

## ✨ Solution

This webhook bridge receives webhooks from Forgejo/Gitea and translates them into Buildkite API calls to trigger builds. It's a simple, stateless HTTP service that runs anywhere and bridges the gap between your self-hosted git server and Buildkite's CI/CD platform.

## 🚀 Features

- ✅ **Simple Setup** - Single binary or Docker container
- ✅ **Stateless** - No database required
- ✅ **Branch-aware** - Forwards branch information correctly
- ✅ **Author tracking** - Preserves commit author information
- ✅ **Multi-pipeline** - One service can handle multiple pipelines
- ✅ **Health checks** - Built-in health endpoint for monitoring
- ✅ **Verbose logging** - Optional detailed logging for debugging
- ✅ **Web UI** - Simple status page with setup instructions

## 📋 Prerequisites

- A Buildkite account with API access
- A Forgejo or Gitea instance
- A server to run the webhook bridge (can be anywhere both services can reach)

## 🔧 Installation

### Option 1: Docker Compose (Recommended)

1. Clone this repository:
```bash
git clone https://github.com/rscorer/buildkite-forgejo-webhook.git
cd buildkite-forgejo-webhook
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Edit `.env` with your credentials:
```env
BUILDKITE_ORG=your-org-name
BUILDKITE_TOKEN=your-api-token-here
```

4. Start the service:
```bash
docker compose up -d
```

### Option 2: Binary

1. Build the binary:
```bash
go build -o webhook-bridge .
```

2. Create a `.env` file with your configuration:
```bash
cp .env.example .env
```

3. Edit `.env` with your credentials:
```env
BUILDKITE_ORG=your-org-name
BUILDKITE_TOKEN=your-api-token-here
```

4. Run the binary:
```bash
./webhook-bridge
```

The application will automatically load configuration from the `.env` file.

## 🔑 Getting a Buildkite API Token

**Important**: The API token must be created for the specific organization you want to use.

1. Log into Buildkite and navigate to your organization (e.g., `https://buildkite.com/your-org`)
2. Go to [Buildkite API Access Tokens](https://buildkite.com/user/api-access-tokens)
3. **Verify** you're in the correct organization (check the organization selector in the top-left)
4. Click "New API Access Token"
5. Give it a descriptive name (e.g., "Forgejo Webhook Bridge")
6. Under **Organization Access**, select your organization from the dropdown
7. Grant the `write_builds` scope
8. Click "Create Token" and copy the token

> ⚠️ **Note**: If you get "No organization found" errors, the token was likely created without selecting an organization. Delete the token and create a new one, ensuring you select your organization in step 6.

## ⚙️ Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BUILDKITE_ORG` | Yes | - | Your Buildkite organization slug |
| `BUILDKITE_TOKEN` | Yes | - | Buildkite API token (requires `write_builds` scope) |
| `WEBHOOK_PORT` | No | `8080` | Port the webhook service listens on |
| `LOG_VERBOSE` | No | `false` | Enable detailed logging for debugging |

## 🔗 Forgejo/Gitea Webhook Setup

For each repository you want to connect:

1. Go to your repository settings in Forgejo/Gitea
2. Navigate to **Settings → Webhooks**
3. Click **Add Webhook** → **Gitea** (or **Forgejo**)
4. Configure the webhook:
   - **Target URL**: `http://your-bridge-host:8080/webhook/<pipeline-slug>`
     - Replace `<pipeline-slug>` with your Buildkite pipeline slug
     - Example: `http://webhook.example.com:8080/webhook/my-app`
   - **HTTP Method**: `POST`
   - **POST Content Type**: `application/json`
   - **Trigger On**: Check **Push events**
   - **Branch filter**: Leave empty (or customize as needed)
   - **Active**: Check this box
5. Click **Add Webhook**
6. Test the webhook using the "Test Delivery" button

### Finding Your Pipeline Slug

Your pipeline slug is in the Buildkite URL:
```
https://buildkite.com/<org-name>/<pipeline-slug>
                                  ^^^^^^^^^^^^^^
```

## 📊 Monitoring

### Health Check Endpoint

The service provides a health check endpoint:
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "org": "your-org-name"
}
```

### Web Interface

Visit `http://localhost:8080/` in your browser to see:
- Service status
- Configuration details
- Setup instructions

### Logs

View logs to monitor webhook activity:
```bash
docker compose logs -f webhook-bridge
```

Example log output:
```
🚀 Buildkite-Forgejo Webhook Bridge v1.0.0
📡 Listening on port 8080
🏢 Buildkite organization: my-org
📨 Webhook: repo=john/my-app, branch=main, commit=abc1234, author=john
✅ Build triggered: my-org/my-app (branch: main, commit: abc1234)
```

## 🔒 Security Considerations

1. **HTTPS**: Run this service behind a reverse proxy (nginx, Caddy, Traefik) with SSL/TLS
2. **Firewall**: Restrict access to only your Forgejo/Gitea IP addresses
3. **Token Security**: Keep your Buildkite API token secure and never commit it to version control
4. **Network**: Run on a private network if possible

Example nginx reverse proxy config:
```nginx
server {
    listen 443 ssl;
    server_name webhook.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🧪 Testing

Test the webhook locally:
```bash
curl -X POST http://localhost:8080/webhook/my-pipeline \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "refs/heads/main",
    "repository": {
      "name": "my-repo",
      "full_name": "user/my-repo"
    },
    "head_commit": {
      "id": "abc123def456",
      "message": "Test commit",
      "author": {
        "name": "John Doe",
        "email": "john@example.com",
        "username": "john"
      }
    },
    "pusher": {
      "username": "john"
    }
  }'
```

## 🐛 Troubleshooting

### Webhook not triggering builds

1. Check the webhook bridge logs: `docker compose logs webhook-bridge`
2. Verify environment variables are set correctly
3. Test the webhook from Forgejo/Gitea using "Test Delivery"
4. Enable verbose logging: `LOG_VERBOSE=true` in `.env`
5. Verify the pipeline slug matches your Buildkite pipeline

### Buildkite API errors

- **401 Unauthorized**: Check your `BUILDKITE_TOKEN` is valid and has `write_builds` scope
- **404 Not Found**: Verify `BUILDKITE_ORG` and pipeline slug are correct
- **422 Unprocessable**: Check the commit SHA exists in your repository

### Connection issues

- Ensure Forgejo/Gitea can reach the webhook bridge (check firewalls, DNS)
- Verify the webhook URL is correct and includes the pipeline slug
- Check the webhook bridge is running: `curl http://localhost:8080/health`

## 🤝 Contributing

Contributions welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

Built to solve the common problem of integrating self-hosted Forgejo/Gitea with Buildkite CI/CD.

Thanks to:
- [Forgejo](https://forgejo.org/) and [Gitea](https://about.gitea.com/) teams for building excellent self-hosted Git platforms
- [Buildkite](https://buildkite.com/) for their flexible CI/CD platform and API

## 📞 Support

- 🐛 [Report issues](https://github.com/rscorer/buildkite-forgejo-webhook/issues)
- 💬 [Discussions](https://github.com/rscorer/buildkite-forgejo-webhook/discussions)
- ⭐ Star this repo if you find it useful!
