import axios from 'axios';

import type { ApiEnvelope } from '../types/api';
import type { Conversation } from '../types/conversation';

interface CreateConversationResult {
  conversationId: string;
}

export class ApiClientError extends Error {
  code: number;
  detail?: unknown;

  constructor(message: string, code: number, detail?: unknown) {
    super(message);
    this.name = 'ApiClientError';
    this.code = code;
    this.detail = detail;
  }
}

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '/api/v1',
  timeout: 5000,
});

function normalizeRequestError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }

  if (axios.isAxiosError(error)) {
    return new Error(error.message);
  }

  return new Error('Unknown request error');
}

function unwrapEnvelope<T>(payload: ApiEnvelope<T>): T {
  if (payload.code !== 0) {
    throw new ApiClientError(payload.message, payload.code, payload.detail);
  }

  return payload.data;
}

export async function createConversation(): Promise<string> {
  try {
    const response = await apiClient.post<ApiEnvelope<CreateConversationResult>>('/conversations');
    return unwrapEnvelope(response.data).conversationId;
  } catch (error) {
    throw normalizeRequestError(error);
  }
}

export async function fetchConversation(conversationId: string): Promise<Conversation> {
  try {
    const response = await apiClient.get<ApiEnvelope<Conversation>>(
      `/conversations/${conversationId}`,
    );
    return unwrapEnvelope(response.data);
  } catch (error) {
    throw normalizeRequestError(error);
  }
}

export async function sendConversationMessage(
  conversationId: string,
  content: string,
): Promise<Conversation> {
  try {
    const response = await apiClient.post<ApiEnvelope<Conversation>>(
      `/conversations/${conversationId}/messages`,
      { content },
    );
    return unwrapEnvelope(response.data);
  } catch (error) {
    throw normalizeRequestError(error);
  }
}
