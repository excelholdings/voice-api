# Voice API

> A high-performance, real-time AI voice assistant API built in Go with Twilio integration for intelligent phone call handling

Voice API is a comprehensive solution for creating AI-powered phone agents that can handle incoming calls, engage in natural conversations, and perform actions based on user requests. Built with Go for performance and reliability, it provides real-time streaming audio processing, intelligent conversation management, and seamless Twilio integration.

## Features

### 🤖 AI-Powered Voice Agents
- **Real-time conversation**: Stream audio processing with minimal latency
- **Multiple LLM support**: OpenAI, Fireworks, and custom model integration
- **Voice synthesis**: ElevenLabs integration for natural-sounding responses
- **Smart endpointing**: Intelligent conversation flow management
- **Multilingual support**: Handle calls in multiple languages

### 📞 Twilio Integration
- **Seamless phone integration**: Direct Twilio WebSocket streaming
- **Call forwarding**: Intelligent call routing and forwarding
- **Voicemail handling**: Automatic voicemail detection and routing
- **Phone number management**: Easy agent phone number setup

### 🛠️ Advanced Features
- **Filler word detection**: Real-time speech analysis and improvement
- **Compliance checks**: Automated content filtering and rewriting
- **Custom actions**: Define custom behaviors and call forwarding
- **Webhook support**: External system integration
- **Analytics**: Detailed call metrics and performance tracking

### 🔒 Enterprise Ready
- **Authentication**: JWT-based user authentication
- **API key management**: Secure API access control
- **Stripe integration**: Subscription and billing management
- **Database persistence**: PostgreSQL with GORM ORM
- **Docker support**: Easy deployment and scaling

## Quick Start

### Prerequisites
- Go 1.22+
- PostgreSQL
- Twilio account
- OpenAI API key (or alternative LLM provider)
- ElevenLabs API key

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd voice-api
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your API keys and configuration
```

4. **Run with Docker Compose**
```bash
docker-compose up -d
```

5. **Start the server**
```bash
go run cmd/main.go
```

### Configuration

Create a `.env` file with the following variables:

```env
# Server Configuration
PORT=8080
ENV=local
JWT_SECRET=your-jwt-secret

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=password
DB_NAME=voice_api

# API Keys
OPENAI_API_KEY=your-openai-key
FIREWORKS_API_KEY=your-fireworks-key
ELEVENLABS_API_KEY=your-elevenlabs-key
DEEPGRAM_API_KEY=your-deepgram-key

# Twilio Configuration
TWILIO_SID=your-twilio-sid
TWILIO_AUTH_TOKEN=your-twilio-auth-token
TWILIO_STREAMING_URL=https://your-domain.com/twilio/stream
TWILIO_ML_SID=your-twilio-ml-sid
TWILIO_ML_URL=https://your-domain.com/twilio/ml

# Payment Processing
STRIPE_SECRET_KEY=your-stripe-secret-key

# Voice Synthesis
CARTESIA_API_KEY=your-cartesia-key
CARTESIA_VERSION=your-cartesia-version
```

## API Usage

### Create an Agent

```bash
curl -X POST http://localhost:8080/v1/agent \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "name": "Customer Service Agent",
    "phone_number": "+1234567890",
    "system_prompt": "You are a helpful customer service representative.",
    "initial_message": "Hello! How can I help you today?",
    "llm_model": "gpt-4",
    "voice_id": "voice-id",
    "filler_words": true,
    "chunking": true,
    "endpointing": 1000
  }'
```

### Handle Incoming Calls

The system automatically handles incoming calls through Twilio:

1. **Call arrives** → Twilio routes to `/twilio/ml`
2. **Stream established** → WebSocket connection to `/twilio/stream`
3. **Real-time processing** → Audio streaming, transcription, and response generation
4. **Call completion** → Recording and analytics stored

### WebSocket Streaming

The system uses WebSocket connections for real-time audio streaming:

```javascript
// Example WebSocket connection for custom clients
const ws = new WebSocket('wss://your-domain.com/twilio/stream');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // Handle audio data, transcriptions, and responses
};
```

## Architecture

### Core Components

- **API Layer** (`internal/api/`): RESTful API endpoints for agent and call management
- **Streaming Layer** (`internal/streaming/`): Real-time audio processing and conversation handling
- **Models** (`internal/models/`): Database models for users, agents, and calls
- **Config** (`internal/config/`): Configuration management
- **Server** (`internal/server/`): HTTP server and routing setup

### Key Features

- **Call Orchestrator**: Manages the entire call flow from start to finish
- **Audio Processing**: Real-time audio streaming with Deepgram integration
- **Conversation Handler**: Manages conversation state and LLM interactions
- **Smart Endpointing**: Intelligent conversation flow control
- **Action Handler**: Executes custom actions based on conversation context

## Development

### Database Migrations

```bash
# Run automatic migrations
go run cmd/main.go db automigrate
```

### Testing

```bash
# Run tests
go test ./...
```

### Building

```bash
# Build for production
go build -o voice-api cmd/main.go
```

## Deployment

### Docker

```bash
# Build and run with Docker
docker build -t voice-api .
docker run -p 8080:8080 voice-api
```

### Docker Compose

```bash
# Full stack deployment
docker-compose up -d
```

### Cloud Deployment

The project includes Cloud Build configurations for Google Cloud Platform deployment:

```bash
# Deploy to GCP
gcloud builds submit --config cloudbuild.yaml
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the API documentation
- Review the OpenAPI specification in `openapi.yaml`