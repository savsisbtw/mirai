import { useRef } from 'preact/hooks';
import { Card } from './Card';
import type { Title } from '../types';

type RowProps = {
  title: string;
  kana: string;
  items: Title[];
  more?: string;
};

export function Row({ title, kana, items, more }: RowProps) {
  const track = useRef<HTMLDivElement>(null);

  const scroll = (direction: number) => {
    const node = track.current;
    if (node) node.scrollBy({ left: direction * node.clientWidth * 0.85, behavior: 'smooth' });
  };

  if (items.length === 0) return null;

  return (
    <section class="row">
      <div class="row-head">
        <span class="kana" aria-hidden="true">{kana}</span>
        <h2>{title}</h2>
        {more && <a class="row-more" href={more}>Все</a>}
        <div class="row-arrows">
          <button type="button" aria-label="Назад" onClick={() => scroll(-1)}>
            <svg width="10" height="10" viewBox="0 0 10 10"><path d="M7 1L3 5l4 4" stroke="currentColor" stroke-width="1.5" fill="none" /></svg>
          </button>
          <button type="button" aria-label="Вперёд" onClick={() => scroll(1)}>
            <svg width="10" height="10" viewBox="0 0 10 10"><path d="M3 1l4 4-4 4" stroke="currentColor" stroke-width="1.5" fill="none" /></svg>
          </button>
        </div>
      </div>
      <div class="row-track" ref={track}>
        {items.map((item) => (
          <Card key={item.id} title={item} />
        ))}
      </div>
    </section>
  );
}
