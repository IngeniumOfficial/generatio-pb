# Generatio PocketBase Extension Tests

This directory contains comprehensive tests for the Generatio PocketBase extension.

## Test Coverage

### Mock FAL Client (`TestMockFALClient`)

- Tests all FAL client operations with configurable mock responses
- Validates token validation, image generation, model retrieval
- Tests error handling for invalid tokens
- Covers queue operations (submit, check status, poll for completion)

### Authentication & Cryptography (`TestAuthAndCrypto`)

- **Encryption Service**: Tests AES-256-GCM encryption/decryption with PBKDF2 key derivation
- **Session Store**: Tests session creation, retrieval, expiration, and deletion
- Validates password-based encryption and secure session management

### API Models (`TestAPIModels`)

- Tests JSON serialization/deserialization of all request/response models
- Validates API error structures
- Ensures proper data structure integrity

### End-to-End Workflow (`TestEndToEndFlow`)

- Simulates complete user journey: token setup → session creation → image generation
- Tests the integration between encryption, session management, and FAL operations
- Validates the encrypted token storage format

### Service Integration (`TestServiceIntegration`)

- Tests encryption service working with session store
- Validates session statistics and cleanup operations
- Tests session expiration behavior

### Complete Workflow (`TestCompleteWorkflow`)

- Full simulation of user registration to image generation
- Tests the entire data flow with proper cleanup
- Validates business logic implementation

## Running Tests

```bash
# Run all tests with verbose output
go test ./tests -v

# Run a specific test
go test ./tests -run TestMockFALClient -v

# Run tests with coverage
go test ./tests -cover
```

## Test Architecture

The tests are designed to work without requiring a full PocketBase database setup by:

1. **Mocking External Services**: FAL client is fully mocked for isolated testing
2. **Testing Business Logic**: Core encryption, session management, and API logic
3. **Validating Data Flow**: End-to-end workflows test the complete system integration
4. **Error Coverage**: Tests include error cases and edge conditions

## Test Data

- Test email: `test@test.com`
- Test password: `testpassword123`
- Test FAL token: `test_fal_token_12345`

All test data is isolated and doesn't affect production systems.

## Notes

- Tests use lower iteration counts for PBKDF2 (1000) to speed up execution
- Session timeouts are shortened for expiration testing
- All sensitive data is properly cleaned up after tests
- Tests validate both success and error paths
