/**
 * SSE (Server-Sent Events) streaming client utility
 *
 * Uses fetch + ReadableStream instead of EventSource to support:
 * - POST method with request body
 * - Custom Authorization headers
 */

export interface SSEOptions {
  /** Full URL for the SSE endpoint */
  url: string;
  /** Bearer token for authentication */
  token?: string;
  /** Request body (will be JSON.stringify'd) */
  body?: Record<string, unknown>;
  /** Called for each data chunk received */
  onChunk: (data: unknown) => void;
  /** Called when stream completes (receives [DONE]) */
  onDone: () => void;
  /** Called on error */
  onError: (error: Error) => void;
}

/**
 * Creates an SSE stream using fetch + ReadableStream
 *
 * @returns AbortController for cancelling the stream
 */
export function createSSEStream(options: SSEOptions): AbortController {
  const { url, token, body, onChunk, onDone, onError } = options;
  const controller = new AbortController();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  fetch(url, {
    method: 'POST',
    headers,
    body: body ? JSON.stringify(body) : undefined,
    signal: controller.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        throw new Error(`SSE request failed: ${response.status} ${response.statusText}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('Response body is not readable');
      }

      const decoder = new TextDecoder();
      let buffer = '';

      try {
        while (true) {
          const { done, value } = await reader.read();

          if (done) {
            break;
          }

          buffer += decoder.decode(value, { stream: true });

          const lines = buffer.split('\n\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed) {
              continue;
            }

            if (trimmed.startsWith('data: ')) {
              const data = trimmed.slice(6);

              if (data === '[DONE]') {
                onDone();
                return;
              }

              try {
                const parsed = JSON.parse(data);
                onChunk(parsed);
              } catch {
                onChunk(data);
              }
            }
          }
        }

        onDone();
      } catch (error) {
        if (error instanceof Error && error.name !== 'AbortError') {
          onError(error);
        }
      }
    })
    .catch((error) => {
      if (error.name !== 'AbortError') {
        onError(error);
      }
    });

  return controller;
}
