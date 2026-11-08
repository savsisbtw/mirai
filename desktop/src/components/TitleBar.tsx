import { useEffect, useState } from 'preact/hooks';
import { getCurrentWindow } from '@tauri-apps/api/window';

const insideTauri = '__TAURI_INTERNALS__' in window;

export function Logo({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <rect x="4" y="4" width="19" height="19" fill="#e4412b" />
      <rect x="1" y="1" width="19" height="19" fill="#0d0d10" stroke="#f3eee3" stroke-width="2" />
      <path d="M5 16V5h2.4l2.6 5 2.6-5H15v11h-2.2v-6.4l-2.8 5-2.8-5V16z" fill="#f3eee3" />
    </svg>
  );
}

export function TitleBar() {
  const [maximized, setMaximized] = useState(false);

  useEffect(() => {
    if (!insideTauri) return;
    const appWindow = getCurrentWindow();
    const refresh = () => appWindow.isMaximized().then(setMaximized);
    refresh();
    const stop = appWindow.onResized(refresh);
    return () => {
      stop.then((unlisten) => unlisten());
    };
  }, []);

  const run = (action: 'minimize' | 'toggleMaximize' | 'close') => {
    if (insideTauri) getCurrentWindow()[action]();
  };

  return (
    <header class="titlebar">
      <div class="titlebar-brand" data-tauri-drag-region>
        <Logo size={16} />
        <span class="titlebar-name" data-tauri-drag-region>MIRAI</span>
        <span class="titlebar-kana" data-tauri-drag-region>ミライ</span>
      </div>
      <div class="titlebar-drag" data-tauri-drag-region />
      <div class="titlebar-buttons">
        <button type="button" aria-label="Свернуть" onClick={() => run('minimize')}>
          <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 5.5h10" stroke="currentColor" stroke-width="1" fill="none" /></svg>
        </button>
        <button type="button" aria-label={maximized ? 'Восстановить' : 'Развернуть'} onClick={() => run('toggleMaximize')}>
          {maximized ? (
            <svg width="10" height="10" viewBox="0 0 10 10"><path d="M2.5 2.5V.5h7v7h-2M.5 2.5h7v7h-7z" stroke="currentColor" stroke-width="1" fill="none" /></svg>
          ) : (
            <svg width="10" height="10" viewBox="0 0 10 10"><path d="M.5.5h9v9h-9z" stroke="currentColor" stroke-width="1" fill="none" /></svg>
          )}
        </button>
        <button type="button" class="close" aria-label="Закрыть" onClick={() => run('close')}>
          <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1" fill="none" /></svg>
        </button>
      </div>
    </header>
  );
}
