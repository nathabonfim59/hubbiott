# Hubbiott

> A Discord bot for PR summaries and media streaming, powered by OpenCode AI

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

Hubbiott is a Discord bot that helps development teams stay informed about Pull Request activity. It generates AI-powered PR summaries using OpenCode and can stream media files directly to Discord channels.

### Key Features

- **AI-Powered PR Summaries** - Get concise, high-level summaries of PRs using OpenCode
- **Configurable Repository Monitoring** - Choose which events trigger summaries (PR opened, merged, etc.)
- **Media Streaming** - Upload and stream media files directly to Discord channels
- **Web Dashboard** - Manage repositories and view activity through an embedded SvelteKit interface
- **GitHub Webhook Integration** - Real-time PR event handling

## Tech Stack

- **Backend**: Go 1.21+ with Echo framework
- **Discord Bot**: discordgo library
- **Database**: SQLite with Turso (libSQL)
- **AI**: OpenCode SDK
- **Frontend**: SvelteKit (embedded)

## Quick Start

### Prerequisites

- Go 1.21 or later
- Node.js 18+ (for frontend development)
- Discord Bot Token
- GitHub Personal Access Token
- Turso account (optional, SQLite works locally)

### Installation

```bash
# Clone the repository
git clone https://github.com/nathabonfim59/hubbiott.git
cd hubbiott

# Copy environment variables
cp .env.example .env

# Edit .env with your credentials
# See Configuration section below

# Run migrations
go run cmd/migrate/main.go

# Build frontend
cd frontend
npm install
npm run build
cd ..

# Run the server
go run cmd/server/main.go
```

### Configuration

Create a `.env` file:

```bash
# Discord Bot
DISCORD_TOKEN=your_discord_bot_token
DISCORD_GUILD_ID=optional_for_testing

# Database (Turso for production, SQLite for local)
DATABASE_URL=libsql://your-db.turso.io
DATABASE_AUTH_TOKEN=your_turso_token
# Or for local development:
# DATABASE_URL=file:./hubbiott.db

# API Security
API_KEY=your_secure_random_api_key

# GitHub
GITHUB_TOKEN=ghp_your_github_token

# OpenCode (optional, SDK uses defaults)
# OPENCODE_API_KEY=your_opencode_key
```

## Usage

### Discord Commands

- `/hubbiott config` - View repository configuration (admin only)

### Web Dashboard

Access the dashboard at `http://localhost:8080` (or your configured port).

Features:
- Add/remove repository monitoring
- Configure PR summary settings per repository
- View recent PR summaries
- Manual PR summary triggers

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/webhooks/github` | POST | GitHub webhook handler |
| `/api/media` | POST | Stream media to Discord |
| `/api/repositories` | GET/POST | List/add repositories |
| `/api/repositories/:id` | DELETE | Remove repository |
| `/api/repositories/:id/summarize` | POST | Manual PR summary |

All API endpoints (except GitHub webhook) require the `X-API-Key` header.

### GitHub Webhook Setup

1. Go to your repository Settings → Webhooks
2. Add webhook: `https://your-domain.com/api/webhooks/github`
3. Content type: `application/json`
4. Secret: Use a secure random string (configure in app)
5. Events: Select "Pull requests"

## Repository Configuration

When adding a repository, you can configure:

- **Discord Channel** - Where summaries are posted
- **AI Model** - Which OpenCode model to use
- **Trigger Events**:
  - On PR opened
  - On PR merged
- **Summary Mode**:
  - `first` - Only the opening message
  - `summary` - Only the AI summary
  - `both` - Both opening message and summary
- **Excluded Authors** - Bot accounts, CI tools, etc. to skip

## Development

### Backend

```bash
# Run with hot reload (requires air)
air

# Or standard go run
go run cmd/server/main.go
```

### Frontend

```bash
cd frontend
npm install
npm run dev        # Development server
npm run build      # Production build
```

### Database Migrations

```bash
# Create new migration
go run cmd/migrate/main.go create migration_name

# Run migrations
go run cmd/migrate/main.go up

# Rollback
go run cmd/migrate/main.go down
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](./LICENSE) file for details.

## Author

**Nathanael Bonfim** ([@nathabonfim59](https://github.com/nathabonfim59))

## Acknowledgments

- [OpenCode SDK](https://pkg.go.dev/github.com/sst/opencode-sdk-go) - AI-powered code analysis
- [discordgo](https://github.com/bwmarrin/discordgo) - Discord Go library
- [Turso](https://turso.tech) - Edge SQLite database
