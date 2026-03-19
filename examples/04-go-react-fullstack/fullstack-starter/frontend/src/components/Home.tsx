import { useState } from 'react';
import type { FormEvent } from 'react';

import type { Conversation } from '../types/conversation';
import { ErrorState } from './common/ErrorState';
import { Loading } from './common/Loading';
import {
  createConversation,
  fetchConversation,
  sendConversationMessage,
} from '../services/api';

function roleClassName(role: 'user' | 'assistant'): string {
  return role === 'user' ? 'message-card message-card-user' : 'message-card message-card-assistant';
}

export function Home() {
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [draft, setDraft] = useState<string>('');
  const [error, setError] = useState<Error | null>(null);
  const [isCreating, setIsCreating] = useState<boolean>(false);
  const [isRefreshing, setIsRefreshing] = useState<boolean>(false);
  const [isSending, setIsSending] = useState<boolean>(false);

  const isBusy = isCreating || isRefreshing || isSending;
  const canRefresh = conversationId !== null && !isBusy;
  const canSend = conversationId !== null && draft.trim() !== '' && !isBusy;

  async function handleCreateConversation() {
    setError(null);
    setIsCreating(true);

    try {
      const nextConversationId = await createConversation();
      const nextConversation = await fetchConversation(nextConversationId);

      setConversationId(nextConversationId);
      setConversation(nextConversation);
      setDraft('');
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError : new Error('Unknown request error'));
    } finally {
      setIsCreating(false);
    }
  }

  async function handleRefreshConversation() {
    if (!conversationId) {
      return;
    }

    setError(null);
    setIsRefreshing(true);

    try {
      const nextConversation = await fetchConversation(conversationId);
      setConversation(nextConversation);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError : new Error('Unknown request error'));
    } finally {
      setIsRefreshing(false);
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!conversationId || draft.trim() === '') {
      return;
    }

    setError(null);
    setIsSending(true);

    try {
      const nextConversation = await sendConversationMessage(conversationId, draft);
      setConversation(nextConversation);
      setDraft('');
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError : new Error('Unknown request error'));
    } finally {
      setIsSending(false);
    }
  }

  return (
    <main className="layout">
      <section className="hero">
        <div>
          <p className="eyebrow">Chapter 4 minimal workbench</p>
          <h1>Conversation starter workspace</h1>
          <p className="hero-copy">
            Create a conversation, send a message, and inspect the full message history through one
            small end-to-end chat loop.
          </p>
        </div>
        <div className="hero-actions">
          <button
            className="primary-button"
            disabled={isBusy}
            onClick={() => void handleCreateConversation()}
            type="button"
          >
            {isCreating ? 'Creating...' : 'Create conversation'}
          </button>
          <button
            className="secondary-button"
            disabled={!canRefresh}
            onClick={() => void handleRefreshConversation()}
            type="button"
          >
            {isRefreshing ? 'Refreshing...' : 'Refresh history'}
          </button>
        </div>
      </section>

      {error ? (
        <ErrorState
          message={error.message}
          onRetry={conversationId ? () => void handleRefreshConversation() : undefined}
        />
      ) : null}

      {isCreating && !conversation ? <Loading message="Creating conversation..." /> : null}

      <section className="chat-shell">
        <aside className="chat-sidebar">
          <div className="status-card">
            <p className="status-label">Conversation</p>
            <strong>{conversationId ?? 'Not created yet'}</strong>
            <p className="status-meta">
              {conversation
                ? `${conversation.messages.length} messages available`
                : 'Create a conversation to begin the chat loop.'}
            </p>
          </div>
          <div className="status-card">
            <p className="status-label">Assistant rule</p>
            <strong>已收到：{'{用户消息}'}</strong>
            <p className="status-meta">
              The backend appends the user message and then generates a deterministic assistant
              reply.
            </p>
          </div>
        </aside>

        <div className="conversation-panel">
          <div className="conversation-header">
            <div>
              <p className="eyebrow">Messages</p>
              <h2>Minimal chat transcript</h2>
            </div>
            <span className="status-pill">{isSending ? 'Sending' : 'Ready'}</span>
          </div>

          <div className="message-list">
            {!conversation ? (
              <div className="empty-state">
                <p>No active conversation yet.</p>
                <p>Create a conversation to start sending messages.</p>
              </div>
            ) : conversation.messages.length === 0 ? (
              <div className="empty-state">
                <p>This conversation is empty.</p>
                <p>Type a message below and the backend will append an assistant reply.</p>
              </div>
            ) : (
              conversation.messages.map((message, index) => (
                <article className={roleClassName(message.role)} key={`${message.role}-${index}`}>
                  <div className="message-meta">
                    <span className="badge badge-info">{message.role}</span>
                  </div>
                  <p>{message.content}</p>
                </article>
              ))
            )}
          </div>

          <form className="composer" onSubmit={(event) => void handleSubmit(event)}>
            <label className="composer-label" htmlFor="message-input">
              Message
            </label>
            <textarea
              className="composer-input"
              disabled={!conversationId || isBusy}
              id="message-input"
              onChange={(event) => setDraft(event.target.value)}
              placeholder={conversationId ? 'Say hello to the assistant...' : 'Create a conversation first'}
              rows={4}
              value={draft}
            />
            <div className="composer-actions">
              <p className="composer-hint">
                {conversationId
                  ? 'Messages are sent to the backend and the full conversation is returned.'
                  : 'Conversation creation is required before sending a message.'}
              </p>
              <button className="primary-button" disabled={!canSend} type="submit">
                {isSending ? 'Sending...' : 'Send message'}
              </button>
            </div>
          </form>
        </div>
      </section>
    </main>
  );
}
