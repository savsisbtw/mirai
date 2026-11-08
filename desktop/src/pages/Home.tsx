import { imageUrl, useApi } from '../api';
import { ErrorBox } from '../components/Common';
import { Hero } from '../components/Hero';
import { Row } from '../components/Row';
import { getHistory } from '../store';
import type { HomeData } from '../types';

function formatTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const rest = Math.floor(seconds % 60);
  return minutes + ':' + String(rest).padStart(2, '0');
}

function Continue() {
  const items = getHistory().slice(0, 8);
  if (items.length === 0) return null;
  return (
    <section class="row">
      <div class="row-head">
        <span class="kana" aria-hidden="true">続き</span>
        <h2>Продолжить просмотр</h2>
        <a class="row-more" href="#/history">История</a>
      </div>
      <div class="row-track">
        {items.map((item) => (
          <a class="resume" key={item.id} href={'#/title/' + item.id + '?play=1'}>
            <div class="resume-frame">
              <img src={imageUrl(item.poster)} width="320" height="180" loading="lazy" decoding="async" alt="" />
              <span class="card-hatch" />
              <span class="resume-ep">{item.ep}</span>
              {item.dur > 0 && (
                <span class="resume-bar">
                  <i style={{ width: Math.min(100, (item.pos / item.dur) * 100) + '%' }} />
                </span>
              )}
            </div>
            <h3 class="card-name">{item.name}</h3>
            <p class="card-meta">Серия {item.ep}{item.pos > 0 ? ' / ' + formatTime(item.pos) : ''}</p>
          </a>
        ))}
      </div>
    </section>
  );
}

export function Home() {
  const { data, error, retry } = useApi<HomeData>('/v1/home');

  if (!data) {
    if (error) return <ErrorBox text={error} retry={retry} />;
    return (
      <div class="page">
        <div class="hero hero-empty" />
      </div>
    );
  }

  return (
    <div class="page">
      <Hero items={data.featured} />
      <Continue />
      <Row title="Новое" kana="新着" items={data.latest} more="#/catalog?sort=new" />
      <Row title="Онгоинги" kana="放送中" items={data.ongoing} more="#/catalog?status=ongoing" />
      <Row title="Популярное" kana="人気" items={data.popular} more="#/catalog?sort=popular" />
    </div>
  );
}
