# 🎨 DoodleDash Backend

A scalable, real-time backend for **DoodleDash**, a multiplayer drawing and guessing game built with Go and WebSocket. Players can create or join rooms, draw collaboratively, guess words, and compete across rounds — all synchronized in real time. This backend powers a fun, interactive experience for web or mobile frontends.

<p align="center">
  <img src="images/doodledash-logo.png" alt="DoodleDash Logo" width="250"/>
</p>

---

## 📚 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Running the Server](#running-the-server)
- [API Endpoints](#api-endpoints)
- [WebSocket Messages](#websocket-messages)
- [Testing](#testing)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

---

## 📌 Overview

DoodleDash is a multiplayer game where players take turns drawing a word while others guess it in real time.

The backend handles:
- Room management
- Game logic
- Real-time drawing
- Scoring

Uses WebSocket for low-latency communication and REST APIs for room operations. It's built to integrate with frontends like **Flutter** or **React**.

---

## 🚀 Features

- **Real-Time Gameplay:** Drawing and guessing synchronized via WebSocket.
- **Room Management:** Create public/private rooms with custom settings.
- **Game Logic:** Multiple rounds, difficulty-based words, hints, scoring.
- **Player Management:** Guest users with avatars and usernames.
- **Security:** Input sanitization, rate limiting, CORS.
- **Scalability:** In-memory room store with cleanup (extendable).
- **Matchmaking:** Smart auto-join for public rooms.

---

## 🛠️ Tech Stack

- **Language:** Go (v1.21+)
- **WebSocket:** [Gorilla WebSocket](https://github.com/gorilla/websocket)
- **Routing:** [Gorilla Mux](https://github.com/gorilla/mux)
- **Config:** YAML via Viper
- **Rate Limiting:** `golang.org/x/time/rate`
- **Logging:** Standard Go logging
- **Dependency Management:** Go Modules

---

## 📦 Prerequisites

- Go 1.21+
- Git
- Any Go-compatible editor (VS Code, GoLand)
- Docker (optional)

---

## ⚙️ Installation

```bash
git clone https://github.com/RITWIZSINGH/DoodleDash-backend.git
cd DoodleDash-backend
go mod tidy
cp configs/config.example.yaml configs/config.yaml
```

---

## 🧾 Configuration

Edit `configs/config.yaml`:

```yaml
server:
  port: ":8080"
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 120s

cors:
  allowed_origins: ["*"]

game:
  max_players_per_room: 8
  round_duration: 60s
  max_rounds: 5

word_bank:
  easy_words_file: "data/words/easy.json"
  medium_words_file: "data/words/medium.json"
  hard_words_file: "data/words/hard.json"
```

---

## 🧱 Project Structure

```
DoodleDash-backend/
├── cmd/server/main.go        # Server entry
├── configs/                  # YAML config
├── data/words/               # Word lists
├── images/                   # Logo and screenshots
├── internal/
│   ├── config/               # Config loader
│   ├── handlers/             # HTTP + WebSocket handlers
│   ├── middleware/           # CORS, rate limiting
│   ├── models/               # User, Room, etc.
│   ├── services/             # Game logic
│   └── websocket/            # Hub + client
├── pkg/utils/                # Utility functions
├── go.mod, go.sum
└── README.md
```

---

## ▶️ Running the Server

```bash
go run cmd/server/main.go
```

Verify:

```bash
curl http://localhost:8080/health
# Output: OK
```

---

## 📡 API Endpoints

| Method | Endpoint              | Description         |
| ------ | --------------------- | ------------------- |
| GET    | `/health`             | Server health check |
| GET    | `/ws`                 | WebSocket upgrade   |
| GET    | `/api/rooms/public`   | List public rooms   |
| POST   | `/api/rooms`          | Create a new room   |
| GET    | `/api/rooms/{roomID}` | Get room info       |

### 🧪 Example Requests

```bash
curl http://localhost:8080/api/rooms/public
```

```bash
curl -X POST -H "Content-Type: application/json" \
-d '{"room_name":"Doodle Room","room_type":"public","max_players":8,"round_time":60,"max_rounds":5,"difficulty":"easy"}' \
http://localhost:8080/api/rooms
```

---

## 🔌 WebSocket Messages

### Client to Server

* `connect`
* `create_room`
* `join_room`
* `start_game`
* `draw_start`
* `send_guess`
* `list_public_rooms`

### Server to Client

* `room_created`
* `game_started`
* `new_round`
* `draw_data`
* `guess_result`
* `round_ended`
* `error`

---

## 🧪 Testing

### Unit Tests:

```bash
go test ./...
```

### Manual:

* Use [wscat](https://github.com/websockets/wscat):

```bash
wscat -c ws://localhost:8080/ws
# Send:
# {"type":"connect","data":{"username":"Player1","avatar":"🎨"}}
```

---

## 🚢 Deployment

### 🧪 Local:

```bash
go build -o doodledash-backend cmd/server/main.go
./doodledash-backend
```

### 🐳 Docker:

Create a `Dockerfile`:

```Dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o doodledash-backend cmd/server/main.go
EXPOSE 8080
CMD ["./doodledash-backend"]
```

Then build and run:

```bash
docker build -t doodledash-backend .
docker run -p 8080:8080 -v $(pwd)/configs:/app/configs doodledash-backend
```

### ☁️ Cloud:

* Deploy to **Heroku**, **AWS**, **Render**, or **DigitalOcean**
* Use HTTPS/WSS in production
* Update `config.yaml` accordingly
* (Optional) Add PostgreSQL or Redis for persistence

---

## 🤝 Contributing

We welcome PRs! 🚀

1. Fork the repo:

   ```bash
   git clone https://github.com/RITWIZSINGH/DoodleDash-backend.git
   ```

2. Create your branch:

   ```bash
   git checkout -b feature/your-feature
   ```

3. Commit and push:

   ```bash
   git push origin feature/your-feature
   ```

4. Open a pull request with a clear description.

---

## 📸 Screenshots

<p align="center">
  <img src="images/gameplay-screenshot.png" alt="Gameplay Screenshot" width="700"/>
</p>

---

## 📄 License

This project is licensed under the MIT License.
See `LICENSE` for details.

---

**Happy doodling!** 🎨
For issues or questions, [open an issue](https://github.com/RITWIZSINGH/DoodleDash-backend/issues).
