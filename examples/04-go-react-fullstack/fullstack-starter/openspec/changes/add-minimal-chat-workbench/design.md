## Context

The starter project already demonstrates a layered Go backend, a typed frontend service layer,
and a shared HTTP response envelope, but the current example only lists static sample data.
There are also unfinished local conversation-oriented files in `backend/internal/domain/conversation`
and `backend/internal/application/chat`, so this change should extend that direction rather than
introducing a parallel pattern.

The requested feature is intentionally small: create a conversation, send one or more user
messages, generate a deterministic assistant reply using the fixed rule `已收到：{用户消息}`,
and render the full message list in the frontend. The implementation must stay in memory,
preserve the `{ code, message, data }` envelope, and add tests across domain, application, and
HTTP layers.

## Goals / Non-Goals

**Goals:**
- Add three versioned conversation endpoints under `/api/v1/conversations`.
- Model conversations and messages as backend domain concepts with validation and ordered history.
- Keep orchestration in an application service that creates conversations, loads conversations,
  appends messages, invokes a deterministic responder, and persists updates in memory.
- Provide a minimal frontend chat workspace that can create a conversation, send a message,
  display conversation history, and surface loading or error states.
- Cover core conversation behavior with automated tests in the domain, application, and HTTP
  layers.

**Non-Goals:**
- Real LLM integration or provider SDKs.
- Streaming or partial assistant responses.
- Database persistence or restart recovery.
- Authentication, authorization, or per-user ownership.
- Complex multi-pane chat UX, conversation list management, or optimistic UI.

## Decisions

### 1. Use an explicit conversation lifecycle

The backend will expose:
- `POST /api/v1/conversations` to create an empty conversation and return `{ conversationId }`
- `POST /api/v1/conversations/:id/messages` to append one user message, generate one assistant
  reply, and return the updated conversation
- `GET /api/v1/conversations/:id` to return the conversation detail and full message list

Rationale:
- This matches the requested flow and keeps client state explicit.
- It avoids hidden side effects where sending a message would silently create missing
  conversations.

Alternatives considered:
- Auto-create on first message send: simpler for clients, but less explicit and conflicts with the
  required create endpoint.

### 2. Keep the domain small and immutable at boundaries

The `conversation` domain package will remain responsible for:
- validating conversation identifiers and non-empty message content
- appending `user` and `assistant` messages in order
- returning copied message slices so callers cannot mutate internal state

The conversation aggregate will not know about HTTP, storage, or response formatting.

Rationale:
- This preserves the current layered structure and keeps the domain pure.
- Copying slices at boundaries matches the repository's Go style requirements around shared
  mutable data.

Alternatives considered:
- Store raw DTOs in the application layer only: faster to write, but weakens the domain boundary
  the starter project is trying to teach.

### 3. Keep assistant reply generation behind an application dependency

The application layer will define a responder dependency and use a fixed-rule implementation such
as `RuleBasedResponder` from infrastructure or a nearby package. The responder will derive the
assistant text from the latest user message and return `已收到：{trimmedMessage}`.

Rationale:
- It keeps orchestration code close to the future shape of a real chat service without pulling in
  a real LLM.
- It allows tests to verify orchestration independently from response generation details.
- It aligns with the existing unfinished `chat.Service` structure instead of replacing it.

Alternatives considered:
- Inline string formatting directly in the application service: slightly smaller, but makes the
  future replacement with a real model client noisier.

### 4. Introduce an in-memory conversation repository keyed by conversation ID

The memory repository will store conversations in a `map[string]conversation.Conversation`
protected by `sync.RWMutex`. It will support create, load, and save operations, returning copies
of stored aggregates when mutation could leak across boundaries.

Rationale:
- In-memory storage satisfies the non-persistence requirement and is consistent with the existing
  sample repository.
- A repository port keeps application tests isolated and leaves room for future database-backed
  storage.

Alternatives considered:
- Package-level map without a repository abstraction: smaller, but violates layering and makes
  testing harder.

### 5. Use typed HTTP DTOs and repository-wide response helpers

The new HTTP handler will:
- decode a typed JSON request body for posting messages
- map domain/application outputs into JSON DTOs shaped for the frontend
- use the shared response envelope helpers
- return stable business error responses for validation failures, missing conversations, and
  unexpected server errors

Expected data payloads:
- create conversation: `{ "conversationId": "conv-..." }`
- get/send conversation: `{ "id": "...", "messages": [{ "role": "user|assistant", "content": "..." }] }`

Rationale:
- The repository already centralizes response formatting in `internal/interfaces/http/response`.
- Explicit DTOs avoid leaking domain internals into transport concerns.

Alternatives considered:
- Return domain structs directly from handlers: simpler short-term, but couples transport shape to
  domain internals and makes envelope changes harder later.

### 6. Keep frontend state local and route API calls through the service layer

The frontend will replace the starter sample list with a minimal chat workbench on the existing
home route. The component will manage:
- the current `conversationId`
- the loaded `conversation`
- the draft message text
- loading states for create and send actions
- the latest error to display and recover from

All network calls will go through `frontend/src/services/`, where typed helpers will interpret the
envelope and normalize non-zero business codes into `ApiClientError`.

Rationale:
- This stays consistent with the repository's frontend conventions.
- The flow is small enough that a dedicated global state library would add noise.

Alternatives considered:
- Extend `useApi` for every chat action: possible, but the chat flow mixes fetch and mutation
  states in a way that is clearer with explicit component state for this minimal screen.

### 7. Add backend tests in the same layers that change

Core tests will cover:
- domain: message validation, trimming, ordering, and defensive copying
- application: create/get/send orchestration, missing conversation behavior, and fixed reply flow
- HTTP: envelope structure, status codes, request validation, and successful message exchange via
  `httptest`

Rationale:
- This matches the requested test scope and the repository testing spec.
- HTTP tests validate the public contract, while domain and application tests keep failures
  localized and easier to diagnose.

Alternatives considered:
- HTTP-only coverage: too broad for fast feedback and misses domain/application regressions.

## Risks / Trade-offs

- [In-memory data is lost on restart] -> Document the limitation clearly and keep the repository
  boundary so persistence can be introduced later without changing handlers.
- [Existing local chat files may diverge from the final API flow] -> Extend and refactor those
  files instead of creating duplicate abstractions, and keep explicit create/get/send methods in
  the application service.
- [Repository test conventions mention Ginkgo while current tests use `testing`] -> Use the
  repository convention as the target for newly touched backend tests, and keep the scope focused
  on the conversation packages and HTTP layer to avoid unrelated churn.
- [Frontend state may feel sparse without conversation history management] -> Keep the UI minimal
  for now and return the full conversation payload after each send so the component can always
  re-render from server truth.

## Migration Plan

- No data migration is required because storage is in memory only.
- Deploying the change adds new routes and replaces the starter home screen content with the chat
  workbench.
- Rollback is straightforward: remove the conversation routes and restore the previous home page.

## Open Questions

- None for implementation. This design assumes missing conversation IDs return a not-found
  business error instead of being auto-created during message send.
