## 1. Backend domain and application flow

- [x] 1.1 Extend conversation domain tests to cover ID validation, message trimming, ordering, and defensive message copies
- [x] 1.2 Refactor the conversation domain and chat application service to support explicit create, get, and send-message operations
- [x] 1.3 Add application-layer tests for creating conversations, loading existing conversations, deterministic assistant replies, validation failures, and missing conversation errors

## 2. Backend infrastructure and HTTP delivery

- [x] 2.1 Implement an in-memory conversation repository and fixed-rule responder, then wire them into the HTTP server container
- [x] 2.2 Add a conversation HTTP handler with typed request and response DTOs, versioned routes, shared envelope helpers, and public handler annotations
- [x] 2.3 Add HTTP tests for `POST /api/v1/conversations`, `POST /api/v1/conversations/:id/messages`, and `GET /api/v1/conversations/:id`, including success, validation, and not-found cases

## 3. Frontend minimal chat workbench

- [x] 3.1 Add typed conversation API models and service-layer functions for create, send-message, and get-conversation flows
- [x] 3.2 Replace the current home screen content with a minimal chat workspace that can create a conversation, send a message, and render the returned message list
- [x] 3.3 Add basic loading, disabled, and error states plus responsive styling for the conversation workbench

## 4. Verification

- [x] 4.1 Run backend tests for domain, application, and HTTP layers and fix any regressions
- [x] 4.2 Run frontend lint and build checks, then verify the full create-send-render flow against the local backend
