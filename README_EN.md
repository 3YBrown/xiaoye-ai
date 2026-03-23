# NanoBanana - AI Content Creation Platform

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)

Open-source multi-modal AI content creation platform. Generate images and videos with Google Gemini, Volcengine Seedream/Seedance, and more.

**Live Demo: [https://xiaoye.io](https://xiaoye.io)**

[English](#features) | [中文](README.md)

## Screenshots

| Generate | Inspiration |
|----------|-------------|
| ![Generate](docs/screenshots/generate.png) | ![Inspiration](docs/screenshots/inspiration.png) |

## Features

- **Image Generation** — Multi-model support (Gemini, Seedream), reference images, resolution up to 4K
- **Video Generation** — Text-to-video & image-to-video (Seedance, Veo 3.1), async task polling
- **E-commerce Batch** — Batch product image generation with templates
- **Prompt Optimization** — AI-powered prompt rewriting via DeepSeek
- **Reverse Prompt** — Extract prompts from existing images
- **Inspiration Feed** — Community gallery with likes, remix, and moderation
- **Credit System** — License key redemption, daily check-in, invite rewards, online payment
- **User System** — Email registration/login, Linux.do OAuth, email binding
- **i18n** — Chinese & English

## Tech Stack

| Layer | Stack |
|-------|-------|
| Frontend | Vue 3, Naive UI, Pinia, Vue Router, Vite |
| Backend | Go (Gin), GORM, MySQL, JWT |
| AI Providers | Google Gemini, Volcengine (Doubao) |
| Storage | Alibaba Cloud OSS |
| Admin | Vue 3 SPA (separate build) |

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Alibaba Cloud OSS bucket (for media storage)
- At least one AI provider API key (Google Gemini or Volcengine)

### Backend

```bash
cd backend
cp .env.example .env    # Edit .env with your configuration
go run main.go          # Starts on :8092
```

### Frontend

```bash
cd frontend
npm install
npm run dev             # Starts on http://localhost:5173
```

### Admin Panel

```bash
cd frontend-admin
npm install
npm run dev             # Starts on http://localhost:5174
```

## Configuration

All configuration is via environment variables. See [README.md](README.md#配置说明) for the full configuration reference.

## Deployment

### Production Build

```bash
# Frontend
cd frontend && npm run build        # Output: frontend/dist/

# Admin
cd frontend-admin && npm run build  # Output: frontend-admin/dist/

# Backend
cd backend && go build -o server main.go
```

Set `CORS_ORIGINS=https://your-domain.com` in your production `.env`.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the [AGPL-3.0](LICENSE). If you deploy a modified version as a network service, you must make the source code available to users of that service.
