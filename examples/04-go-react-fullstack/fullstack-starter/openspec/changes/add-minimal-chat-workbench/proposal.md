## Why

The current starter project demonstrates layered backend and typed frontend requests, but it
still lacks a concrete end-to-end interaction loop. Adding a minimal conversation workbench
creates the first reusable chat-shaped workflow that later chapters can extend toward real AI
features.

## What Changes

- Add backend conversation APIs under `/api/v1/conversations` for creating a conversation,
  appending a user message, generating a fixed assistant reply, and fetching conversation
  details.
- Introduce an in-memory conversation repository plus domain and application behavior for
  conversation creation, message appending, and deterministic assistant replies.
- Add a minimal frontend chat workspace that can create a conversation, send a message, and
  render the full message list with basic loading and error states.
- Keep the shared API response envelope `{ code, message, data }` for all new endpoints and
  return the updated conversation payload after sending a message.
- Add tests for domain rules, application orchestration, and HTTP request-response behavior
  around the new conversation flow.

## Capabilities

### New Capabilities
- `conversation-workbench`: Minimal conversation creation, message exchange, and message history
  display backed by deterministic in-memory chat behavior.

### Modified Capabilities
- None.

## Impact

- Backend domain, application, in-memory storage, HTTP handlers, route wiring, and tests.
- Frontend components, service layer, shared API types, and UI state handling.
- New repository-owned API endpoints:
  `POST /api/v1/conversations`,
  `POST /api/v1/conversations/:id/messages`,
  `GET /api/v1/conversations/:id`.
