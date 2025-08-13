# Generatio PocketBase Extension

A PocketBase extension for AI image generation using FAL AI, with encrypted token storage and session management.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PocketBase    │    │   Encryption    │    │   FAL Client    │
│   Database      │◄──►│   Service       │◄──►│   API Calls     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Session Store (In-Memory)                    │
└─────────────────────────────────────────────────────────────────┘
                               ▲
                               │
                               ▼
                    ┌─────────────────┐
                    │   HTTP APIs     │
                    └─────────────────┘
```

### Core Components

- **PocketBase Database**: User data, generated images, preferences, folders
- **Encryption Service**: AES-256-GCM with PBKDF2 for FAL token security
- **Session Store**: In-memory session management with automatic cleanup
- **FAL Client**: Async image generation with queue/polling system

## Installation

```bash
go build -o generatio-pb
./generatio-pb serve --http="127.0.0.1:8090"
```

## Database Setup

Access PocketBase admin at `http://127.0.0.1:8090/_/` and create these collections:

### Generatio Users Collection (auth type)

**Collection Name:** `generatio_users`
**Type:** auth

Add fields to the auth collection:

- `fal_token` (text) - Encrypted FAL AI token with salt (format: "encrypted.salt")
- `financial_data` (json) - Spending tracking data
- `model_preferences` (relation) - Relation to model_preferences collection

### Images Collection

**Collection Name:** `images`

```json
{
  "name": "images",
  "type": "base",
  "fields": [
    { "name": "title", "type": "text" },
    { "name": "url", "type": "text", "required": true },
    { "name": "user_id", "type": "relation", "required": true },
    { "name": "prompt", "type": "text", "required": true },
    { "name": "request_id", "type": "text" },
    { "name": "model", "type": "text", "required": true },
    { "name": "batch_number", "type": "number" },
    { "name": "image_size", "type": "json" },
    { "name": "other_info", "type": "json" },
    { "name": "folder_id", "type": "relation" },
    { "name": "deleted_at", "type": "date" }
  ]
}
```

### Folders Collection

**Collection Name:** `folders`

```json
{
  "name": "folders",
  "type": "base",
  "fields": [
    { "name": "name", "type": "text", "required": true },
    { "name": "user_id", "type": "relation", "required": true },
    { "name": "parent_id", "type": "relation" },
    { "name": "private", "type": "bool" },
    { "name": "deleted_at", "type": "date" }
  ]
}
```

### Model Preferences Collection

**Collection Name:** `model_preferences`

```json
{
  "name": "model_preferences",
  "type": "base",
  "fields": [
    { "name": "model_name", "type": "text", "required": true },
    { "name": "preferences", "type": "json", "required": true }
  ]
}
```

## API Endpoints

All endpoints require PocketBase authentication unless noted.

### System Status

#### `GET /api/custom/status`

Returns system status and available models.

**Response:**

```json
{
  "status": "ok",
  "services": {
    "encryption": "AES-256-GCM with PBKDF2",
    "sessions": {
      "active": 0,
      "total": 0
    }
  },
  "available_models": {
    "flux/schnell": {
      "name": "flux/schnell",
      "display_name": "Flux Schnell",
      "cost_per_image": 0.003
    },
    "hidream/hidream-i1-dev": {
      "name": "hidream/hidream-i1-dev",
      "display_name": "Hi-Dream I1 Dev",
      "cost_per_image": 0.004
    }
  }
}
```

### Token Management

#### `POST /api/custom/tokens/setup`

Encrypt and store FAL AI token.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Request:**

```json
{
  "fal_token": "fal-ai-token-here",
  "password": "encryption-password"
}
```

**Response:**

```json
{
  "success": true,
  "message": "FAL token setup successfully"
}
```

#### `POST /api/custom/tokens/verify`

Verify stored token accessibility.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Request:**

```json
{
  "password": "encryption-password"
}
```

**Response:**

```json
{
  "has_token": true,
  "can_decrypt": true
}
```

### Session Management

#### `POST /api/custom/auth/create-session`

Create in-memory session with decrypted FAL token.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Request:**

```json
{
  "password": "encryption-password"
}
```

**Response:**

```json
{
  "session_id": "uuid-session-id",
  "expires_at": "2024-01-01T12:00:00Z"
}
```

#### `DELETE /api/custom/auth/session`

Delete active session.

**Headers:**

- `Authorization: Bearer <pocketbase_jwt>`
- `X-Session-ID: <session_id>`

**Response:**

```json
{
  "success": true,
  "message": "Session deleted successfully"
}
```

### Image Generation

#### `POST /api/custom/generate/image`

Generate AI images using FAL API.

**Headers:**

- `Authorization: Bearer <pocketbase_jwt>`
- `X-Session-ID: <session_id>`

**Request:**

```json
{
  "model": "flux/schnell",
  "prompt": "A beautiful sunset over mountains",
  "parameters": {
    "image_size": "square_hd",
    "num_images": 1,
    "guidance_scale": 7.5
  },
  "collection_id": "optional-folder-id"
}
```

**Response:**

```json
{
  "images": [
    {
      "id": "generated-image-id",
      "url": "https://fal.ai/generated-image.jpg",
      "thumbnail_url": "https://fal.ai/thumb.jpg"
    }
  ],
  "cost": 0.003,
  "model": "flux/schnell"
}
```

#### `GET /api/custom/generate/models`

List available AI models and their parameters.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Response:**

```json
{
  "flux/schnell": {
    "name": "flux/schnell",
    "display_name": "Flux Schnell",
    "description": "Fast, high-quality image generation",
    "cost_per_image": 0.003,
    "parameters": {
      "image_size": {
        "type": "string",
        "default": "square_hd",
        "options": ["square_hd", "square", "portrait_4_3", "landscape_4_3"],
        "description": "Image size preset"
      },
      "num_images": {
        "type": "integer",
        "default": 1,
        "min": 1,
        "max": 4,
        "description": "Number of images to generate"
      }
    }
  }
}
```

### Financial Tracking

#### `GET /api/custom/financial/stats`

Get spending statistics.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Response:**

```json
{
  "total_spent": 0.25,
  "total_images": 83,
  "recent_spending": 0.05,
  "average_cost": 0.003
}
```

### User Preferences

#### `GET /api/custom/preferences/{model_name}`

Get saved preferences for a model.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Response:**

```json
{
  "model_name": "flux/schnell",
  "has_preferences": true,
  "preferences": {
    "image_size": "square_hd",
    "guidance_scale": 7.5
  }
}
```

#### `POST /api/custom/preferences/{model_name}`

Save preferences for a model.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Request:**

```json
{
  "preferences": {
    "image_size": "square_hd",
    "guidance_scale": 7.5,
    "num_inference_steps": 4
  }
}
```

**Response:**

```json
{
  "success": true,
  "message": "Preferences saved successfully"
}
```

### Collections Management

#### `POST /api/custom/collections/create`

Create image folder/collection.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Request:**

```json
{
  "name": "My Collection",
  "parent_id": "optional-parent-folder-id"
}
```

**Response:**

```json
{
  "id": "folder-id",
  "name": "My Collection",
  "parent_id": "parent-id",
  "created": "2024-01-01T12:00:00Z"
}
```

#### `GET /api/custom/collections`

List user folders/collections.

**Headers:** `Authorization: Bearer <pocketbase_jwt>`

**Response:**

```json
{
  "collections": [
    {
      "id": "folder-id",
      "user_id": "user-id",
      "name": "My Collection",
      "parent_id": "",
      "private": false,
      "created": "2024-01-01T12:00:00Z",
      "updated": "2024-01-01T12:00:00Z"
    }
  ]
}
```

## Security Features

- **Zero-knowledge encryption**: Server never sees plaintext FAL tokens
- **AES-256-GCM encryption** with PBKDF2 key derivation (100,000 iterations)
- **Combined salt storage**: Encrypted data and salt stored as "encrypted.salt" format
- **In-memory sessions**: No persistent session storage
- **Multi-layer authentication**: PocketBase JWT + session validation
- **Input validation**: All parameters validated against model requirements
- **Automatic cleanup**: Background session cleanup and expired data removal

## Supported AI Models

- **flux/schnell**: Fast generation, $0.003 per image
- **hidream/hidream-i1-dev**: High quality, $0.004 per image
- **hidream/hidream-i1-fast**: Fast quality, $0.003 per image

## Development

### Project Structure

```
├── main.go                         # Application entry point
├── internal/
│   ├── auth/
│   │   ├── sessions.go             # Session management
│   │   └── cleanup.go              # Background cleanup
│   ├── crypto/
│   │   └── encryption.go           # AES-256-GCM encryption
│   ├── fal/
│   │   ├── client.go               # FAL AI client
│   │   ├── mock_client.go          # Mock client for testing
│   │   ├── interface.go            # FAL client interface
│   │   └── models.go               # Model definitions
│   ├── handlers/
│   │   ├── handlers.go             # Base handler and routing
│   │   ├── auth_handlers.go        # Authentication endpoints
│   │   ├── generation_handlers.go  # Image generation endpoints
│   │   ├── user_handlers.go        # User management endpoints
│   │   ├── collections_handlers.go # Collections management
│   │   └── example.go              # Example/testing endpoints
│   ├── models/
│   │   └── types.go                # Data structures and API models
│   └── utils/
│       ├── validation.go           # Input validation
│       └── errors.go               # Error handling
├── tests/
│   ├── api_test.go                 # Comprehensive API tests
│   └── README.md                   # Test documentation
└── README.md
```

### Testing

The project includes comprehensive test coverage:

```bash
# Run all tests with verbose output
go test ./tests -v

# Run tests with coverage
go test ./tests -cover
```

**Test Coverage:**

- Mock FAL client operations
- Encryption/decryption workflows
- Session management and cleanup
- API request/response validation
- Complete business logic flows
- Error handling and edge cases

### Configuration

- **HTTP Address**: `127.0.0.1:8090`
- **Session Timeout**: 24 hours
- **Cleanup Interval**: 1 hour
- **PBKDF2 Iterations**: 100,000
- **FAL Timeout**: 10 minutes

### Building

```bash
go mod tidy
go build -o generatio-pb
./generatio-pb serve
```

## Error Handling

All endpoints return standardized error responses:

```json
{
  "error": "validation_error",
  "message": "Detailed error message"
}
```

**Error Codes:**

- `validation_error`: Invalid input data
- `authentication_error`: Missing/invalid authentication
- `authorization_error`: Insufficient permissions
- `not_found`: Resource not found
- `internal_error`: Server error
- `external_error`: FAL AI service error
- `rate_limit_error`: Rate limit exceeded

## Technical Implementation

### Encryption Details

- **Algorithm**: AES-256-GCM with PBKDF2-SHA256
- **Key Derivation**: 100,000 iterations, 32-byte salt
- **Storage Format**: Combined "encrypted_data.base64_salt" format
- **Zero-Knowledge**: Server never accesses plaintext tokens

### Session Management

- **Storage**: In-memory with automatic cleanup
- **Timeout**: Configurable (default 24 hours)
- **Security**: Session IDs are UUIDs, tokens cleared on deletion
- **Cleanup**: Background goroutine removes expired sessions

### Database Integration

- **Primary Collection**: `generatio_users` (auth type)
- **Image Storage**: `images` collection with relation to users
- **Organization**: `folders` collection for image organization
- **Preferences**: `model_preferences` collection for user settings
- **Soft Deletes**: Uses `deleted_at` timestamps for recovery

This implementation provides a secure, scalable foundation for AI image generation with comprehensive testing and proper separation of concerns.
