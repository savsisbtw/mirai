import type { Brief, HistoryItem, Settings } from './types';

function read<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : fallback;
  } catch {
    return fallback;
  }
}

function write(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    return;
  }
}

const defaultSettings: Settings = {
  server: 'http://127.0.0.1:8787',
  autoSkip: false,
  quality: '720'
};

let settings: Settings = { ...defaultSettings, ...read<Partial<Settings>>('mirai.settings', {}) };

export function getSettings(): Settings {
  return settings;
}

export function saveSettings(patch: Partial<Settings>) {
  settings = { ...settings, ...patch };
  write('mirai.settings', settings);
}

export function toBrief(title: Brief): Brief {
  return { id: title.id, name: title.name, poster: title.poster, year: title.year, type: title.type };
}

export function getFavorites(): Brief[] {
  return read<Brief[]>('mirai.favorites', []);
}

export function isFavorite(id: string): boolean {
  return getFavorites().some((item) => item.id === id);
}

export function toggleFavorite(title: Brief): boolean {
  const list = getFavorites();
  const exists = list.some((item) => item.id === title.id);
  const next = exists ? list.filter((item) => item.id !== title.id) : [toBrief(title), ...list];
  write('mirai.favorites', next);
  return !exists;
}

export function getHistory(): HistoryItem[] {
  return read<HistoryItem[]>('mirai.history', []);
}

export function getHistoryItem(id: string): HistoryItem | undefined {
  return getHistory().find((item) => item.id === id);
}

export function saveHistory(item: HistoryItem) {
  const rest = getHistory().filter((entry) => entry.id !== item.id);
  write('mirai.history', [item, ...rest].slice(0, 100));
}

export function removeHistory(id: string) {
  write('mirai.history', getHistory().filter((item) => item.id !== id));
}

export function clearHistory() {
  write('mirai.history', []);
}

export function clearFavorites() {
  write('mirai.favorites', []);
}

export function readFlag(key: string): boolean {
  return read<boolean>(key, false);
}

export function writeFlag(key: string, value: boolean) {
  write(key, value);
}
