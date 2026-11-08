import type { ComponentChildren } from 'preact';

export function ErrorBox({ text, retry }: { text: string; retry: () => void }) {
  return (
    <div class="notice notice-error">
      <p class="notice-title">Не получилось загрузить</p>
      <p class="notice-text">{text}</p>
      <button type="button" class="btn" onClick={retry}>Повторить</button>
    </div>
  );
}

export function Empty({ title, text }: { title: string; text?: string }) {
  return (
    <div class="notice">
      <p class="notice-title">{title}</p>
      {text && <p class="notice-text">{text}</p>}
    </div>
  );
}

export function PageHead({ kana, title, children }: { kana: string; title: string; children?: ComponentChildren }) {
  return (
    <div class="page-head">
      <span class="kana" aria-hidden="true">{kana}</span>
      <h1>{title}</h1>
      {children}
    </div>
  );
}
