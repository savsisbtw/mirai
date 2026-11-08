import { useState } from 'preact/hooks';
import { PageHead } from '../components/Common';
import { clearFavorites, clearHistory, getSettings, saveSettings } from '../store';

export function Settings() {
  const current = getSettings();
  const [server, setServer] = useState(current.server);
  const [autoSkip, setAutoSkip] = useState(current.autoSkip);
  const [quality, setQuality] = useState(current.quality);
  const [message, setMessage] = useState('');

  const saveServer = async () => {
    const address = server.trim().replace(/\/+$/, '');
    try {
      const response = await fetch(address + '/v1/status', { signal: AbortSignal.timeout(5000) });
      if (!response.ok) throw new Error();
    } catch {
      setMessage('Сервер по этому адресу не отвечает. Адрес не сохранён.');
      return;
    }
    saveSettings({ server: address });
    location.reload();
  };

  return (
    <div class="page settings">
      <PageHead kana="設定" title="Настройки" />

      <section class="panel">
        <h2>Сервер</h2>
        <p class="panel-text">Адрес каталога. По умолчанию приложение запускает свой сервер на этом компьютере.</p>
        <div class="panel-row">
          <input
            class="field field-wide"
            type="text"
            value={server}
            onInput={(event) => setServer((event.target as HTMLInputElement).value)}
          />
          <button type="button" class="btn btn-primary" onClick={saveServer}>Проверить и сохранить</button>
        </div>
        {message && <p class="panel-error">{message}</p>}
      </section>

      <section class="panel">
        <h2>Плеер</h2>
        <label class="check">
          <input
            type="checkbox"
            checked={autoSkip}
            onChange={(event) => {
              const value = (event.target as HTMLInputElement).checked;
              setAutoSkip(value);
              saveSettings({ autoSkip: value });
            }}
          />
          <span>Автоматически пропускать опенинг и эндинг</span>
        </label>
        <div class="panel-row">
          <span class="label">Качество по умолчанию</span>
          <select
            class="field"
            value={quality}
            onChange={(event) => {
              const value = (event.target as HTMLSelectElement).value;
              setQuality(value);
              saveSettings({ quality: value });
            }}
          >
            <option value="480">480p</option>
            <option value="720">720p</option>
            <option value="1080">1080p</option>
          </select>
        </div>
      </section>

      <section class="panel">
        <h2>Данные</h2>
        <p class="panel-text">Избранное и история хранятся только на этом компьютере.</p>
        <div class="panel-row">
          <button
            type="button"
            class="btn"
            onClick={() => {
              clearHistory();
              setMessage('История очищена.');
            }}
          >
            Очистить историю
          </button>
          <button
            type="button"
            class="btn"
            onClick={() => {
              clearFavorites();
              setMessage('Избранное очищено.');
            }}
          >
            Очистить избранное
          </button>
        </div>
      </section>
    </div>
  );
}
