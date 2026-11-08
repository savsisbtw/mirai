import { useEffect, useRef, useState } from 'preact/hooks';
import { imageUrl, serverUrl, useApi } from '../api';
import { ErrorBox } from '../components/Common';
import { HlsPlayer } from '../components/HlsPlayer';
import type { Route } from '../router';
import { getHistoryItem, isFavorite, saveHistory, toggleFavorite } from '../store';
import type { Episode, TitleDetail, Voice } from '../types';

const statusLabels: Record<string, string> = {
  ongoing: 'Онгоинг',
  released: 'Вышло',
  announce: 'Анонс'
};

function pad(value: string): string {
  return /^\d+$/.test(value) ? value.padStart(2, '0') : value;
}

function Body({ title, route }: { title: TitleDetail; route: Route }) {
  const history = getHistoryItem(title.id);
  const wantedVoice = route.query.get('voice') || history?.voice;
  const firstVoice = title.voices.find((voice) => voice.id === wantedVoice) || title.voices[0];
  const wantedEpisode = route.query.get('ep') || history?.ep;

  const [voiceId, setVoiceId] = useState(firstVoice?.id || '');
  const [episodeN, setEpisodeN] = useState(() => {
    const found = firstVoice?.episodes.find((item) => item.n === wantedEpisode);
    return found ? found.n : firstVoice?.episodes[0]?.n || '';
  });
  const [started, setStarted] = useState(route.query.get('play') === '1');
  const [favorite, setFavorite] = useState(isFavorite(title.id));
  const player = useRef<HTMLDivElement>(null);

  const voice: Voice | undefined = title.voices.find((item) => item.id === voiceId);
  const episodes = voice?.episodes || [];
  const index = episodes.findIndex((item) => item.n === episodeN);
  const episode: Episode | undefined = episodes[index];

  useEffect(() => {
    if (started) player.current?.scrollIntoView({ block: 'nearest' });
  }, [started, voiceId, episodeN]);

  const record = (position: number, duration: number) => {
    saveHistory({
      id: title.id,
      name: title.name,
      poster: title.poster,
      year: title.year,
      type: title.type,
      ep: episodeN,
      voice: voiceId,
      pos: position,
      dur: duration,
      at: Date.now()
    });
  };

  const pickEpisode = (n: string) => {
    setEpisodeN(n);
    setStarted(true);
  };

  const pickVoice = (next: Voice) => {
    setVoiceId(next.id);
    if (!next.episodes.some((item) => item.n === episodeN)) setEpisodeN(next.episodes[0]?.n || '');
  };

  useEffect(() => {
    if (started && episode && episode.url) record(0, 0);
  }, [started, voiceId, episodeN]);

  const goNext = () => {
    const next = episodes[index + 1];
    if (next) setEpisodeN(next.n);
  };

  const resumeAt = history && history.voice === voiceId && history.ep === episodeN && history.dur - history.pos > 30 ? history.pos : 0;

  return (
    <div class="page title-page">
      <section class="title-head">
        <div class="title-poster">
          <img src={imageUrl(title.poster)} width="240" height="360" alt="" decoding="async" />
          <span class="card-tone" />
        </div>
        <div class="title-info">
          <p class="title-stamp">
            <span class="kana" aria-hidden="true">作品</span>
            <span>{title.sources.join(' + ')}</span>
          </p>
          <h1>{title.name}</h1>
          <p class="title-en">{[title.name_en, ...title.name_alt.slice(0, 2)].filter(Boolean).join(' / ')}</p>
          <ul class="tags">
            <li class="tag tag-red">{statusLabels[title.status]}</li>
            {title.year > 0 && <li class="tag">{title.year}</li>}
            <li class="tag">{title.type}</li>
            {title.season && <li class="tag">{title.season}</li>}
            {title.age && <li class="tag">{title.age}</li>}
            {title.episodes && <li class="tag">{title.episodes} эп.</li>}
            {title.rating > 0 && <li class="tag">★ {title.rating.toFixed(1)}</li>}
          </ul>
          <ul class="tags tags-genres">
            {title.genres.map((genre) => (
              <li class="tag tag-sticker" key={genre}>{genre}</li>
            ))}
          </ul>
          <p class="title-desc">{title.description || 'Описания пока нет.'}</p>
          <div class="hero-actions">
            {episode && (
              <button type="button" class="btn btn-primary" onClick={() => setStarted(true)}>
                {history ? 'Продолжить, серия ' + pad(episodeN) : 'Смотреть'}
              </button>
            )}
            <button
              type="button"
              class={favorite ? 'btn btn-on' : 'btn'}
              onClick={() => setFavorite(toggleFavorite(title))}
            >
              {favorite ? 'В избранном' : 'В избранное'}
            </button>
          </div>
        </div>
      </section>

      {title.voices.length === 0 && (
        <div class="notice">
          <p class="notice-title">Видео пока нет</p>
          <p class="notice-text">Ни один из источников не отдал серии для этого тайтла.</p>
        </div>
      )}

      {voice && (
        <section class="watch" ref={player}>
          <div class="section-head">
            <span class="kana" aria-hidden="true">視聴</span>
            <h2>Просмотр</h2>
          </div>
          <div class="player-wrap">
            {started && episode ? (
              voice.kind === 'hls' ? (
                <HlsPlayer
                  key={voice.id + '/' + episode.n}
                  episode={episode}
                  startAt={resumeAt}
                  hasNext={index < episodes.length - 1}
                  onProgress={record}
                  onNext={goNext}
                />
              ) : (
                <iframe
                  key={voice.id + '/' + episode.n}
                  class="embed"
                  src={serverUrl() + '/v1/embed?u=' + encodeURIComponent(episode.url || '')}
                  allow="autoplay; fullscreen"
                  allowFullScreen
                  referrerPolicy="origin"
                />
              )
            ) : (
              <button type="button" class="player-idle" onClick={() => setStarted(true)} disabled={!episode}>
                <img src={imageUrl(title.backdrop || title.poster)} alt="" decoding="async" />
                <span class="player-idle-play">
                  <svg width="28" height="28" viewBox="0 0 28 28"><path d="M7 3l18 11L7 25z" fill="currentColor" /></svg>
                </span>
              </button>
            )}
          </div>

          <div class="voices">
            <span class="label">Озвучка</span>
            {title.voices.map((item) => (
              <button
                type="button"
                key={item.id}
                class={item.id === voiceId ? 'voice on' : 'voice'}
                onClick={() => pickVoice(item)}
              >
                {item.label}
                <small>{item.kind === 'hls' ? 'HLS' : item.player}</small>
              </button>
            ))}
          </div>

          <div class="episodes">
            {episodes.map((item) => (
              <button
                type="button"
                key={item.n}
                class={item.n === episodeN ? 'ep on' : 'ep'}
                title={item.title || ''}
                onClick={() => pickEpisode(item.n)}
              >
                <b>{pad(item.n)}</b>
                <span>{item.title || ' '}</span>
              </button>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

export function TitlePage({ route }: { route: Route }) {
  const { data, error, retry } = useApi<TitleDetail>('/v1/title/' + route.id);

  if (!data) {
    if (error) return <ErrorBox text={error} retry={retry} />;
    return (
      <div class="page title-page">
        <section class="title-head">
          <div class="title-poster card-empty" />
          <div class="title-info" />
        </section>
      </div>
    );
  }

  return <Body key={data.id} title={data} route={route} />;
}
