import { useEffect, useState } from 'preact/hooks';
import { imageUrl, prefetch } from '../api';
import type { TitleFull } from '../types';

const statusLabels: Record<string, string> = {
  ongoing: 'Онгоинг',
  released: 'Вышло',
  announce: 'Анонс'
};

const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

export function Hero({ items }: { items: TitleFull[] }) {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (reducedMotion || paused || items.length < 2) return;
    const timer = setInterval(() => setIndex((value) => (value + 1) % items.length), 7000);
    return () => clearInterval(timer);
  }, [paused, items.length, index]);

  if (items.length === 0) return null;

  return (
    <section
      class="hero"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      {items.map((item, position) => (
        <article class={position === index ? 'hero-slide on' : 'hero-slide'} key={item.id} aria-hidden={position !== index}>
          <div class="hero-text">
            <p class="hero-stamp">
              <span>№ {pad(position + 1)}</span>
              <span class="hero-kana" aria-hidden="true">注目</span>
            </p>
            <h1>{item.name}</h1>
            <p class="hero-en">{item.name_en}</p>
            <ul class="tags">
              <li class="tag tag-red">{statusLabels[item.status]}</li>
              {item.year > 0 && <li class="tag">{item.year}</li>}
              <li class="tag">{item.type}</li>
              {item.genres.slice(0, 3).map((genre) => (
                <li class="tag" key={genre}>{genre}</li>
              ))}
            </ul>
            <p class="hero-desc">{item.description}</p>
            <div class="hero-actions">
              <a
                class="btn btn-primary"
                href={'#/title/' + item.id + '?play=1'}
                tabIndex={position === index ? 0 : -1}
                onMouseEnter={() => prefetch('/v1/title/' + item.id)}
              >
                Смотреть
              </a>
              <a class="btn" href={'#/title/' + item.id} tabIndex={position === index ? 0 : -1}>Подробнее</a>
            </div>
          </div>
          <div class="hero-art">
            <div class="hero-poster">
              <img
                src={imageUrl(item.backdrop || item.poster)}
                width="400"
                height="600"
                alt=""
                loading={position === 0 ? 'eager' : 'lazy'}
                decoding="async"
              />
              <span class="card-tone" />
            </div>
          </div>
        </article>
      ))}
      <nav class="hero-nav" aria-label="Подборка">
        {items.map((item, position) => (
          <button
            type="button"
            key={item.id}
            class={position === index ? 'on' : ''}
            onClick={() => setIndex(position)}
          >
            {pad(position + 1)}
          </button>
        ))}
      </nav>
    </section>
  );
}
