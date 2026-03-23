# NanoBanana Admin Frontend

Separated admin frontend for inspiration moderation.

## Local development

```bash
npm install
npm run dev
```

Default URL: `http://localhost:5174`

Backend must run on `http://localhost:8092` and provide:

- `GET /api/admin/inspirations`
- `POST /api/admin/inspirations/:id/review`

The UI sends `X-Admin-Token` and stores it as `admin_token` in `localStorage`.
