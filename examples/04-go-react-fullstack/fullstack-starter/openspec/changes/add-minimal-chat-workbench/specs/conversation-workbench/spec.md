## ADDED Requirements

### Requirement: The system MUST create empty conversations through a versioned API
The system MUST allow clients to create an empty conversation through a repository-owned API and
return the created identifier inside the shared response envelope.

#### Scenario: Creating a conversation successfully
- **WHEN** the client sends `POST /api/v1/conversations`
- **THEN** the server MUST return HTTP 200
- **AND** the JSON body MUST include `code`, `message`, and `data`
- **AND** `code` MUST be `0`
- **AND** `message` MUST be `success`
- **AND** `data.conversationId` MUST contain the created conversation identifier

### Requirement: The system MUST append user messages and deterministic assistant replies
The system MUST accept a user message for an existing conversation, append the user message,
generate one assistant reply using the rule `已收到：{用户消息}`, append that reply, and return
the updated conversation state.

#### Scenario: Sending a message to an existing conversation
- **WHEN** the client sends `POST /api/v1/conversations/:id/messages` with a non-empty message
- **THEN** the server MUST append the trimmed user message to the target conversation
- **AND** the server MUST append one assistant message whose content equals
  `已收到：{trimmedUserMessage}`
- **AND** the server MUST return the updated conversation in the shared response envelope
- **AND** the returned message list MUST preserve message order

#### Scenario: Sending a message to a missing conversation
- **WHEN** the client sends `POST /api/v1/conversations/:id/messages` for a conversation ID that
  does not exist
- **THEN** the server MUST return a not-found error using the shared response envelope

#### Scenario: Sending an empty message
- **WHEN** the client sends `POST /api/v1/conversations/:id/messages` with empty or whitespace-only
  content
- **THEN** the server MUST reject the request with an error response using the shared response
  envelope

### Requirement: The system MUST expose conversation details and full message history
The system MUST allow clients to fetch a conversation by identifier and receive the full ordered
message list for that conversation.

#### Scenario: Fetching an existing conversation
- **WHEN** the client sends `GET /api/v1/conversations/:id`
- **THEN** the server MUST return HTTP 200 with the shared response envelope
- **AND** the response data MUST include the conversation identifier
- **AND** the response data MUST include the full ordered message list for that conversation

#### Scenario: Fetching a missing conversation
- **WHEN** the client sends `GET /api/v1/conversations/:id` for a conversation ID that does not
  exist
- **THEN** the server MUST return a not-found error using the shared response envelope

### Requirement: The frontend MUST provide a minimal conversation workbench
The frontend MUST provide a chat workspace that can create a conversation, send a message through
the shared service layer, render the returned message history, and surface basic loading and error
states.

#### Scenario: Creating and using a conversation from the UI
- **WHEN** the user opens the home page and creates a conversation
- **THEN** the UI MUST store the returned conversation ID
- **AND** the UI MUST enable sending a message for that conversation
- **AND** after a successful send the UI MUST render both the user message and the assistant reply

#### Scenario: Showing asynchronous state in the UI
- **WHEN** the frontend is creating a conversation or sending a message
- **THEN** the UI MUST display a loading state for the active action
- **AND** if the request fails the UI MUST display an error state without crashing the page
