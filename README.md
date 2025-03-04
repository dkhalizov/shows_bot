# TV Shows Notification Bot

<div align="center">

![License](https://img.shields.io/github/license/deniskhalizov/shows_bot)
![Go Report Card](https://goreportcard.com/badge/github.com/deniskhalizov/shows_bot)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/deniskhalizov/shows_bot/go.yml?branch=main)
![Go Version](https://img.shields.io/badge/Go-1.24-blue)

A Telegram bot that helps users track their favorite TV shows and receive notifications about upcoming episodes.

[Features](#features) • [Getting Started](#getting-started) • [Installation](#installation) • [Usage](#usage) • [Configuration](#configuration) • [Contributing](#contributing) • [License](#license)

</div>

## 📋 Features

- **Show Search**: Find TV shows using TMDB and TVMaze APIs
- **Show Details**: View comprehensive information about TV shows including status, air dates, and descriptions
- **Episode Tracking**: Follow shows to receive updates about upcoming episodes
- **Automated Notifications**: Get notified about new episodes of followed shows
- **User-friendly Interface**: Simple menu-based navigation with inline buttons

## 🛠️ Tech Stack

- **Language**: Go (v1.24)
- **Database**: PostgreSQL
- **APIs**:
  - [TMDB (The Movie Database) API](https://developers.themoviedb.org/3)
  - [TVMaze API](https://www.tvmaze.com/api)
- **Bot Framework**: [Telegram Bot API](https://core.telegram.org/bots/api)
- **Containerization**: Docker with multi-platform support (linux/amd64, linux/arm64)
- **CI/CD**: GitHub Actions for automated builds and deployment

## 🚀 Getting Started

### Prerequisites

- Go 1.24 or higher
- PostgreSQL database
- Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- TMDB API Key (optional, but recommended)

### Quick Start

1. Clone the repository
   ```bash
   git clone https://github.com/deniskhalizov/shows_bot.git
   cd shows_bot
   ```

2. Copy the example environment file and configure your settings
   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

3. Run with Docker
   ```bash
   docker-compose up -d
   ```

   Or build and run locally
   ```bash
   make build
   make run
   ```

## 📥 Installation

### Environment Variables

The following environment variables need to be set:

```env
TELEGRAM_TOKEN=your_telegram_bot_token
DATABASE_URL=postgres://username:password@hostname:port/database
TMDB_API_KEY=your_tmdb_api_key  # Optional
```

### Running Locally

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Set up the database:
   ```bash
   make migrate
   ```

3. Run the bot:
   ```bash
   go run cmd/main.go
   ```

### Docker Deployment

The project includes a Dockerfile for containerized deployment:

1. Build the Docker image:
   ```bash
   docker build -t tv-shows-bot .
   ```

2. Run the container:
   ```bash
   docker run -d --name tv-shows-bot \
     -e TELEGRAM_TOKEN=your_telegram_bot_token \
     -e DATABASE_URL=postgres://username:password@hostname:port/database \
     -e TMDB_API_KEY=your_tmdb_api_key \
     tv-shows-bot
   ```

### Docker Compose

For local development with all dependencies, use Docker Compose:

```bash
docker-compose up -d
```

### GitHub Actions CI/CD

The repository is configured with GitHub Actions workflows for:

- Building the Docker image
- Running tests
- Publishing to GitHub Container Registry (ghcr.io)
- Multi-platform builds (linux/amd64, linux/arm64)

To deploy from GitHub Container Registry:

```bash
docker pull ghcr.io/deniskhalizov/shows_bot:latest
```

## 📱 Usage

Once the bot is running, users can interact with it via Telegram:

1. Start the bot by sending `/start`
2. Use the menu buttons to navigate or type commands:
   - `/search [query]` - Search for TV shows
   - `/list` - View followed shows
   - `/upcoming` - Check upcoming episodes
   - `/help` - Get help and instructions

### Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Initialize the bot and display welcome message |
| `/help` | Show help information |
| `/search [query]` | Search for TV shows by name |
| `/list` | Show your followed shows |
| `/upcoming` | Display upcoming episodes for followed shows |

### Screenshots

<!-- Add screenshots here when available -->

## ⚙️ Configuration

The bot can be configured via the `config.yaml` file or environment variables:

```yaml
# Core credentials
telegram_token: ${TELEGRAM_TOKEN}
database_url: ${DATABASE_URL}

# API keys for external services
api_keys:
  tmdb: ${TMDB_API_KEY}

# Bot-specific settings
bot:
  name: "TV Shows Notification Bot"
  notification_enabled: true
  check_interval: 6h
  max_results: 5
  max_followed_shows: 100
  episode_notification_threshold: 24h
```

See the [full configuration guide](docs/configuration.md) for more details.

## 📂 Project Structure

```
├── .github
│   └── workflows      # GitHub Actions CI/CD workflows
├── clients            # API client implementations
│   ├── tmdb           # TMDB API client
│   └── tvmaze         # TVMaze API client
├── cmd
│   └── main.go        # Application entry point
├── internal
│   ├── bot            # Telegram bot implementation
│   ├── config         # Configuration handling
│   ├── database       # Database operations
│   └── models         # Data models
├── migrations         # Database migrations
├── Dockerfile         # Docker build instructions
├── docker-compose.yml # Local development setup
├── go.mod             # Go module definition
└── README.md          # Project documentation
```

## 🛡️ API Clients

The bot uses two TV show data providers:

1. **TMDB Client**: Fetches show information, episodes, and images from The Movie Database
2. **TVMaze Client**: Provides alternative show data and scheduling information

The clients implement a common interface (`ShowAPIClient`), making it easy to add more providers in the future.

## 🗄️ Database Schema

The application uses PostgreSQL with the following tables:

- `users`: Stores Telegram user information
- `shows`: Contains TV show details
- `episodes`: Stores episode information
- `user_shows`: Tracks which users follow which shows
- `notifications`: Records which notifications have been sent

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- [TMDB API](https://www.themoviedb.org/documentation/api) for TV show data
- [TVMaze API](https://www.tvmaze.com/api) for additional show information
- [Telegram Bot API](https://core.telegram.org/bots/api) for the messaging platform
- [Go Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api) for the Telegram client library
