import { useState } from 'preact/hooks';
import { imageUrl } from '../api';
import { Empty, PageHead } from '../components/Common';
import { clearFavorites, clearHistory, getFavorites, getHistory, removeHistory, toggleFavorite } from '../store';

export function Favorites() {
  const [items, setItems] = useState(getFavorites());

  return (
    <div class="page">
      <PageHead kana="お気に入り" title="Избранное">
        {items.length > 0 && (
          <button
            type="button"
            class="btn btn-small"
            onClick={() => {
              clearFavorites();
              setItems([]);
            }}
          >
            Очистить
          </button>
        )}
      </PageHead>
      {items.length === 0 ? (
        <Empty title="Здесь пусто" text="Добавляйте тайтлы кнопкой «В избранное» на странице тайтла." />
      ) : (
        <div class="grid">
          {items.map((item) => (
            <div class="card-wrap" key={item.id}>
              <a class="card" href={'#/title/' + item.id}>
                <div class="card-frame">
                  <img src={imageUrl(item.poster)} width="200" height="300" loading="lazy" decoding="async" alt="" />
                  <span class="card-hatch" />
                  <span class="card-tone" />
                </div>
                <h3 class="card-name">{item.name}</h3>
                <p class="card-meta">{[item.year || '', item.type].filter(Boolean).join(' / ')}</p>
              </a>
              <button
                type="button"
                class="card-remove"
                aria-label="Убрать из избранного"
                onClick={() => {
                  toggleFavorite(item);
                  setItems(getFavorites());
                }}
              >
                <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1.4" fill="none" /></svg>
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function History() {
  const [items, setItems] = useState(getHistory());

  return (
    <div class="page">
      <PageHead kana="履歴" title="История">
        {items.length > 0 && (
          <button
            type="button"
            class="btn btn-small"
            onClick={() => {
              clearHistory();
              setItems([]);
            }}
          >
            Очистить
          </button>
        )}
      </PageHead>
      {items.length === 0 ? (
        <Empty title="Вы пока ничего не смотрели" />
      ) : (
        <div class="grid">
          {items.map((item) => (
            <div class="card-wrap" key={item.id}>
              <a class="card" href={'#/title/' + item.id + '?play=1'}>
                <div class="card-frame">
                  <img src={imageUrl(item.poster)} width="200" height="300" loading="lazy" decoding="async" alt="" />
                  <span class="card-hatch" />
                  <span class="card-tone" />
                  <span class="card-badge">Серия {item.ep}</span>
                  {item.dur > 0 && (
                    <span class="resume-bar">
                      <i style={{ width: Math.min(100, (item.pos / item.dur) * 100) + '%' }} />
                    </span>
                  )}
                </div>
                <h3 class="card-name">{item.name}</h3>
                <p class="card-meta">{new Date(item.at).toLocaleDateString('ru-RU')}</p>
              </a>
              <button
                type="button"
                class="card-remove"
                aria-label="Убрать из истории"
                onClick={() => {
                  removeHistory(item.id);
                  setItems(getHistory());
                }}
              >
                <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1.4" fill="none" /></svg>
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
