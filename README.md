# Generatio PocketBase Extension

A comprehensive PocketBase extension for the Generatio AI image generation application, featuring zero-knowledge encryption, secure session management, and FAL AI integration.

## Features

### 🔐 Security

- **Zero-knowledge encryption**: AES-256-GCM with PBKDF2 key derivation for FAL AI tokens
- **Pure in-memory sessions**: No disk persistence for maximum security
- **Multi-layer authentication**: PocketBase JWT + session validation + resource ownership
- **Comprehensive input validation**: Sanitization and validation utilities

### 🎨 AI Integration

- **FAL AI client**: Queue/polling system for async image generation
- **Multiple models**: Support for flux/schnell, hidream variants
- **Cost tracking**: Precise financial monitoring and spending statistics
- **Parameter validation**: Model-specific parameter validation

### 📊 Data Management

- **Extended user collections**: Encrypted token storage, preferences, financial data
- **Generated images**: Full metadata and cost tracking
- **Collections management**: User-organized image collections
- **Model preferences**: Per-model parameter presets

## Quick Start

### 1. Build and Run

```bash
# Build the application
go build

# Run the server
./myapp serve --http="127.0.0.1:8090"
```

### 2. Setup Database Collections

Access the PocketBase admin interface at `http://127.0.0.1:8090/_/` and create the following collections:

#### Users Collection (extend existing)

Add these fields to the existing users collection:

- `fal_token` (text) - Encrypted FAL AI token
- `salt` (text) - Encryption salt for the token
- `financial_data` (json) - Spending statistics and limits

#### Generated Images Collection

```json
{
  "name": "generated_images",
  "fields": [
    {
      "name": "user_id",
      "type": "relation",
      "options": { "collectionId": "users" }
    },
    { "name": "model_name", "type": "text" },
    { "name": "prompt", "type": "text" },
    { "name": "image_url", "type": "url" },
    { "name": "parameters", "type": "json" },
    { "name": "cost_usd", "type": "number" },
    { "name": "generation_time_ms", "type": "number" },
    { "name": "fal_request_id", "type": "text" }
  ]
}
```

#### Model Preferences Collection

```json
{
  "name": "model_preferences",
  "fields": [
    {
      "name": "user_id",
      "type": "relation",
      "options": { "collectionId": "users" }
    },
    { "name": "model_name", "type": "text" },
    { "name": "parameters", "type": "json" }
  ]
}
```

#### Collections Collection

```json
{
  "name": "collections",
  "fields": [
    {
      "name": "user_id",
      "type": "relation",
      "options": { "collectionId": "users" }
    },
    { "name": "name", "type": "text" },
    { "name": "description", "type": "text" },
    { "name": "image_ids", "type": "json" }
  ]
}
```

## API Endpoints

### Example/Testing Endpoints

#### Health Check

```bash
GET /api/custom/status
```

Returns system status, active sessions, and available models.

#### Test Encryption

```bash
POST /api/custom/test/encryption
Content-Type: application/json

{
  "text": "Hello World",
  "password": "test123"
}
```

Tests the encryption/decryption functionality.

### Production Endpoints (Planned)

#### Token Management

- `POST /api/custom/tokens/setup` - Setup encrypted FAL AI token
- `POST /api/custom/tokens/verify` - Verify token validity

#### Session Management

- `POST /api/custom/auth/create-session` - Create secure session
- `DELETE /api/custom/auth/session` - Delete session

#### Image Generation

- `POST /api/custom/generate/image` - Generate image with cost tracking
- `GET /api/custom/generate/models` - Get available models

#### Financial Tracking

- `GET /api/custom/financial/stats` - Get spending statistics

#### User Preferences

- `GET /api/custom/preferences/{model_name}` - Get model preferences
- `POST /api/custom/preferences/{model_name}` - Save model preferences

#### Collections Management

- `POST /api/custom/collections/create` - Create image collection
- `GET /api/custom/collections` - List user collections

## Architecture

### Core Components

1. **Encryption Service** (`internal/crypto/`)

   - AES-256-GCM encryption with PBKDF2 key derivation
   - 100,000 PBKDF2 iterations for security
   - Cryptographically secure random salt generation

2. **Session Management** (`internal/auth/`)

   - Pure in-memory session store
   - Automatic cleanup of expired sessions
   - Thread-safe operations with mutex protection

3. **FAL AI Integration** (`internal/fal/`)

   - Async queue/polling system
   - Model parameter validation
   - Cost calculation and tracking

4. **Validation & Error Handling** (`internal/utils/`)
   - Comprehensive input sanitization
   - Standardized error responses
   - Security-focused validation rules

### Security Model

- **Zero-knowledge encryption**: Server never sees plaintext FAL tokens
- **Session isolation**: Each user session is completely isolated
- **Memory-only sessions**: No session data persisted to disk
- **Multi-layer auth**: PocketBase JWT + session + ownership validation
- **Input sanitization**: All user inputs validated and sanitized

## Development

### Project Structure

```
.
├── main.go                     # Application entry point
├── internal/
│   ├── auth/                   # Session management
│   │   ├── sessions.go         # Session store implementation
│   │   └── cleanup.go          # Background cleanup service
│   ├── crypto/                 # Encryption utilities
│   │   └── encryption.go       # AES-256-GCM implementation
│   ├── fal/                    # FAL AI integration
│   │   ├── client.go           # FAL AI client
│   │   └── models.go           # Model definitions
│   ├── handlers/               # HTTP handlers
│   │   └── example.go          # Example endpoints
│   ├── models/                 # Data types
│   │   └── types.go            # Shared types
│   └── utils/                  # Utilities
│       ├── validation.go       # Input validation
│       └── errors.go           # Error handling
├── ARCHITECTURE.md             # Detailed architecture
├── IMPLEMENTATION_PLAN.md      # Implementation roadmap
├── SECURITY_ANALYSIS.md        # Security analysis
└── PROJECT_SUMMARY.md          # Project overview
```

### Building

```bash
# Install dependencies
go mod tidy

# Build
go build

# Run tests (when implemented)
go test ./...
```

### Configuration

The application uses PocketBase's standard configuration. Key settings:

- **HTTP Address**: `127.0.0.1:8090` (default)
- **Session Timeout**: 24 hours
- **Cleanup Interval**: 1 hour
- **PBKDF2 Iterations**: 100,000
- **FAL Timeout**: 10 minutes

## Security Considerations

1. **Token Storage**: FAL AI tokens are encrypted with user passwords and never stored in plaintext
2. **Session Security**: Sessions are memory-only and automatically cleaned up
3. **Input Validation**: All inputs are validated and sanitized
4. **Error Handling**: Errors don't leak sensitive information
5. **Authentication**: Multi-layer authentication ensures proper access control

## License

This project is part of the Generatio application suite.
