import { useEffect, useState } from 'preact/hooks';
import { getSettings } from './store';

type CacheEntry = { data: unknown; at: number };

const cache = new Map<string, CacheEntry>();
const pending = new Map<string, Promise<unknown>>();
const freshForMs = 30000;

export function serverUrl(): string {
  return getSettings().server.replace(/\/+$/, '');
}

export function imageUrl(path: string): string {
  if (!path) return '';
  if (path.startsWith('http') || path.startsWith('data:')) return path;
  return serverUrl() + path;
}

export async function fetchJson<T>(path: string): Promise<T> {
  const response = await fetch(serverUrl() + path, { signal: AbortSignal.timeout(20000) });
  const body = await response.json().catch(() => null);
  if (!response.ok) throw new Error(body?.error || 'Ошибка сервера ' + response.status);
  return body as T;
}

export function get<T>(path: string): Promise<T> {
  const running = pending.get(path);
  if (running) return running as Promise<T>;
  const request = fetchJson<T>(path)
    .then((data) => {
      cache.set(path, { data, at: Date.now() });
      return data;
    })
    .finally(() => pending.delete(path));
  pending.set(path, request);
  return request;
}

export function cached<T>(path: string): T | undefined {
  return cache.get(path)?.data as T | undefined;
}

export function prefetch(path: string) {
  if (!cache.has(path)) get(path).catch(() => undefined);
}

type State<T> = { path: string | null; data?: T; error?: string };

export function useApi<T>(path: string | null) {
  const [state, setState] = useState<State<T>>({ path, data: path ? cached<T>(path) : undefined });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!path) return;
    let alive = true;
    const hit = cache.get(path);
    setState({ path, data: hit?.data as T | undefined });
    if (hit && Date.now() - hit.at < freshForMs) return;
    get<T>(path)
      .then((data) => alive && setState({ path, data }))
      .catch((error: Error) => alive && setState((old) => ({ path, data: old.data, error: error.message })));
    return () => {
      alive = false;
    };
  }, [path, attempt]);

  const data = state.path === path ? state.data : path ? cached<T>(path) : undefined;
  const error = state.path === path && !data ? state.error : undefined;
  return { data, error, retry: () => setAttempt((value) => value + 1) };
}
