# end to end encrypted chat system

## Usage

### Live
Visit **[zukichat.online](https://zukichat.online/)** — no setup required.

> Messages are stored server-side for **7 days** only.

### Self-host

1. Create a `.env` file with the following variables:
   - `CHAT_DB_PASSWORD`
   - `JWT_SECRET`
   - `GATEWAY_PUBLIC_URL`  (optional, for GitHub sign-in)
   - `GITHUB_OAUTH_CLIENT_ID` / `GITHUB_OAUTH_CLIENT_SECRET` (optional, for GitHub sign-in)

2. Start the stack:
   ```
   docker compose up -d
   ```

3. Open `http://localhost` in your browser.
