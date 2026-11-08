import { imageUrl, prefetch } from '../api';
import { memo } from '../memo';
import type { Title } from '../types';

const statusLabels: Record<string, string> = {
  ongoing: 'Онгоинг',
  released: 'Вышло',
  announce: 'Анонс'
};

let hoverTimer = 0;

function startPrefetch(id: string) {
  clearTimeout(hoverTimer);
  hoverTimer = window.setTimeout(() => prefetch('/v1/title/' + id), 150);
}

function stopPrefetch() {
  clearTimeout(hoverTimer);
}

type CardProps = {
  title: Title;
  badge?: string;
};

function CardView({ title, badge }: CardProps) {
  return (
    <a
      class="card"
      href={'#/title/' + title.id}
      onMouseEnter={() => startPrefetch(title.id)}
      onMouseLeave={stopPrefetch}
    >
      <div class="card-frame">
        <img src={imageUrl(title.poster)} width="200" height="300" loading="lazy" decoding="async" alt="" />
        <span class="card-hatch" />
        <span class="card-tone" />
        {title.rating > 0 && <span class="card-rating">{title.rating.toFixed(1)}</span>}
        {badge ? (
          <span class="card-badge">{badge}</span>
        ) : (
          title.status === 'ongoing' && <span class="card-badge">{statusLabels.ongoing}</span>
        )}
      </div>
      <h3 class="card-name">{title.name}</h3>
      <p class="card-meta">
        {[title.year || '', title.type].filter(Boolean).join(' / ')}
      </p>
    </a>
  );
}

export const Card = memo(CardView);

export function CardPlaceholders({ count }: { count: number }) {
  return (
    <div class="grid">
      {Array.from({ length: count }, (_, index) => (
        <div class="card card-empty" key={index}>
          <div class="card-frame" />
          <h3 class="card-name">&nbsp;</h3>
        </div>
      ))}
    </div>
  );
}
