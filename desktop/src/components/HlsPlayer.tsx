import { useEffect, useRef, useState } from 'preact/hooks';
import { getSettings, saveSettings } from '../store';
import type { Episode } from '../types';

type PlayerProps = {
  episode: Episode;
  startAt: number;
  hasNext: boolean;
  onProgress: (position: number, duration: number) => void;
  onNext: () => void;
};

function pickQuality(qualities: string[], wanted: string): string {
  const sorted = [...qualities].sort((a, b) => Number(a) - Number(b));
  const fitting = sorted.filter((item) => Number(item) <= Number(wanted));
  return fitting.length > 0 ? fitting[fitting.length - 1] : sorted[0];
}

function formatTime(seconds: number): string {
  if (!isFinite(seconds)) return '0:00';
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const rest = String(total % 60).padStart(2, '0');
  if (hours > 0) return hours + ':' + String(minutes).padStart(2, '0') + ':' + rest;
  return minutes + ':' + rest;
}

export function HlsPlayer({ episode, startAt, hasNext, onProgress, onNext }: PlayerProps) {
  const box = useRef<HTMLDivElement>(null);
  const video = useRef<HTMLVideoElement>(null);
  const resumeAt = useRef(startAt);
  const lastSave = useRef(0);
  const skipped = useRef(new Set<string>());
  const hideTimer = useRef(0);
  const latest = useRef({ onProgress, onNext, hasNext, episode });
  latest.current = { onProgress, onNext, hasNext, episode };

  const qualities = Object.keys(episode.hls || {});
  const [quality, setQuality] = useState(() => pickQuality(qualities, getSettings().quality));
  const [attempt, setAttempt] = useState(0);
  const [error, setError] = useState('');
  const [playing, setPlaying] = useState(false);
  const [time, setTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [autoSkip, setAutoSkip] = useState(getSettings().autoSkip);
  const [skipKind, setSkipKind] = useState<'opening' | 'ending' | null>(null);
  const [full, setFull] = useState(false);
  const [controls, setControls] = useState(true);

  useEffect(() => {
    resumeAt.current = startAt;
    skipped.current = new Set();
    setQuality((old) => (qualities.includes(old) ? old : pickQuality(qualities, getSettings().quality)));
  }, [episode]);

  useEffect(() => {
    const url = episode.hls?.[quality];
    if (!url) {
      setError('Для этой серии нет потока');
      return;
    }
    setError('');
    let dead = false;
    let hls: { destroy: () => void } | null = null;
    import('hls.js').then(({ default: Hls }) => {
      const element = video.current;
      if (dead || !element) return;
      if (!Hls.isSupported()) {
        element.src = url;
        element.currentTime = resumeAt.current;
        element.play().catch(() => undefined);
        return;
      }
      const player = new Hls({ startPosition: resumeAt.current, maxBufferLength: 40 });
      hls = player;
      player.loadSource(url);
      player.attachMedia(element);
      player.on(Hls.Events.MANIFEST_PARSED, () => {
        element.play().catch(() => undefined);
      });
      player.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) setError('Не удалось загрузить видео');
      });
    });
    return () => {
      dead = true;
      hls?.destroy();
    };
  }, [episode, quality, attempt]);

  useEffect(() => {
    return () => {
      const element = video.current;
      if (element && element.currentTime > 0) latest.current.onProgress(element.currentTime, element.duration || 0);
    };
  }, [episode]);

  const toggle = () => {
    const element = video.current;
    if (!element) return;
    if (element.paused) element.play().catch(() => undefined);
    else element.pause();
  };

  const seekBy = (delta: number) => {
    const element = video.current;
    if (element) element.currentTime = Math.max(0, Math.min(element.duration || 0, element.currentTime + delta));
  };

  const changeVolume = (delta: number) => {
    const element = video.current;
    if (!element) return;
    element.muted = false;
    element.volume = Math.max(0, Math.min(1, element.volume + delta));
  };

  const toggleMute = () => {
    const element = video.current;
    if (element) element.muted = !element.muted;
  };

  const toggleFull = () => {
    if (document.fullscreenElement) document.exitFullscreen();
    else box.current?.requestFullscreen();
  };

  const changeQuality = (value: string) => {
    resumeAt.current = video.current?.currentTime || 0;
    saveSettings({ quality: value });
    setQuality(value);
  };

  const toggleAutoSkip = () => {
    saveSettings({ autoSkip: !autoSkip });
    setAutoSkip(!autoSkip);
  };

  const skip = (kind: 'opening' | 'ending') => {
    const element = video.current;
    const range = kind === 'opening' ? episode.opening : episode.ending;
    if (!element || !range) return;
    if (kind === 'ending' && range.stop >= (element.duration || 0) - 2) onNext();
    else element.currentTime = range.stop;
  };

  const handleTime = () => {
    const element = video.current;
    if (!element) return;
    const now = element.currentTime;
    setTime(now);
    if (Date.now() - lastSave.current > 5000) {
      lastSave.current = Date.now();
      onProgress(now, element.duration || 0);
    }
    const opening = episode.opening;
    const ending = episode.ending;
    let kind: 'opening' | 'ending' | null = null;
    if (opening && now >= opening.start && now < opening.stop) kind = 'opening';
    if (ending && now >= ending.start && now < ending.stop) kind = 'ending';
    setSkipKind((old) => (old === kind ? old : kind));
    if (kind && getSettings().autoSkip && !skipped.current.has(kind)) {
      skipped.current.add(kind);
      skip(kind);
    }
  };

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (target.tagName === 'TEXTAREA' || target.tagName === 'SELECT' || (target.tagName === 'INPUT' && (target as HTMLInputElement).type !== 'range')) return;
      if (event.code === 'Space') toggle();
      else if (event.code === 'ArrowLeft') seekBy(event.shiftKey ? -30 : -5);
      else if (event.code === 'ArrowRight') seekBy(event.shiftKey ? 30 : 5);
      else if (event.code === 'ArrowUp') changeVolume(0.05);
      else if (event.code === 'ArrowDown') changeVolume(-0.05);
      else if (event.code === 'KeyF') toggleFull();
      else if (event.code === 'KeyM') toggleMute();
      else return;
      event.preventDefault();
    };
    const onFull = () => setFull(document.fullscreenElement === box.current);
    addEventListener('keydown', onKey);
    document.addEventListener('fullscreenchange', onFull);
    return () => {
      removeEventListener('keydown', onKey);
      document.removeEventListener('fullscreenchange', onFull);
    };
  }, []);

  const wake = () => {
    setControls(true);
    clearTimeout(hideTimer.current);
    hideTimer.current = window.setTimeout(() => setControls(false), 2800);
  };

  const showControls = controls || !playing;

  return (
    <div class={showControls ? 'player' : 'player idle'} ref={box} onMouseMove={wake} onMouseLeave={() => playing && setControls(false)}>
      <video
        ref={video}
        class="player-video"
        onClick={toggle}
        onDblClick={toggleFull}
        onPlay={() => {
          setPlaying(true);
          wake();
        }}
        onPause={() => {
          setPlaying(false);
          const element = video.current;
          if (element) onProgress(element.currentTime, element.duration || 0);
        }}
        onEnded={() => hasNext && onNext()}
        onTimeUpdate={handleTime}
        onLoadedMetadata={() => setDuration(video.current?.duration || 0)}
        onDurationChange={() => setDuration(video.current?.duration || 0)}
        onVolumeChange={() => {
          setVolume(video.current?.volume ?? 1);
          setMuted(video.current?.muted ?? false);
        }}
      />
      {error && (
        <div class="player-error">
          <p>{error}</p>
          <button type="button" class="btn" onClick={() => setAttempt((value) => value + 1)}>Повторить</button>
        </div>
      )}
      {skipKind && (
        <button type="button" class="player-skip" onClick={() => skip(skipKind)}>
          {skipKind === 'opening' ? 'Пропустить опенинг' : 'Пропустить эндинг'}
        </button>
      )}
      <div class="player-bar">
        <input
          class="player-seek"
          type="range"
          min="0"
          max={duration || 0}
          step="0.5"
          value={time}
          aria-label="Перемотка"
          style={{ '--filled': duration ? (time / duration) * 100 + '%' : '0%' }}
          onInput={(event) => {
            const element = video.current;
            if (element) element.currentTime = Number((event.target as HTMLInputElement).value);
          }}
        />
        <div class="player-row">
          <button type="button" onClick={toggle} aria-label={playing ? 'Пауза' : 'Играть'}>
            {playing ? (
              <svg width="14" height="14" viewBox="0 0 14 14"><path d="M3 1h3v12H3zM8 1h3v12H8z" fill="currentColor" /></svg>
            ) : (
              <svg width="14" height="14" viewBox="0 0 14 14"><path d="M3 1l10 6-10 6z" fill="currentColor" /></svg>
            )}
          </button>
          <span class="player-time">{formatTime(time)} / {formatTime(duration)}</span>
          <span class="player-gap" />
          <button type="button" class={autoSkip ? 'player-toggle on' : 'player-toggle'} onClick={toggleAutoSkip} title="Автоматически пропускать опенинг и эндинг">
            Авто-пропуск
          </button>
          <select
            class="player-quality"
            value={quality}
            aria-label="Качество"
            onChange={(event) => changeQuality((event.target as HTMLSelectElement).value)}
          >
            {[...qualities].sort((a, b) => Number(a) - Number(b)).map((item) => (
              <option key={item} value={item}>{item}p</option>
            ))}
          </select>
          <button type="button" onClick={toggleMute} aria-label="Звук">
            <svg width="16" height="14" viewBox="0 0 16 14">
              <path d="M1 5h3l4-3.5v11L4 9H1z" fill="currentColor" />
              {muted || volume === 0 ? (
                <path d="M10 4l5 6M15 4l-5 6" stroke="currentColor" stroke-width="1.4" fill="none" />
              ) : (
                <path d="M10.5 4.5q2 2.5 0 5M12.5 2.5q3.5 4.5 0 9" stroke="currentColor" stroke-width="1.2" fill="none" />
              )}
            </svg>
          </button>
          <button type="button" onClick={toggleFull} aria-label="Во весь экран">
            {full ? (
              <svg width="14" height="14" viewBox="0 0 14 14"><path d="M5 1v4H1M9 1v4h4M5 13V9H1M9 13V9h4" stroke="currentColor" stroke-width="1.3" fill="none" /></svg>
            ) : (
              <svg width="14" height="14" viewBox="0 0 14 14"><path d="M1 5V1h4M9 1h4v4M1 9v4h4M13 9v4H9" stroke="currentColor" stroke-width="1.3" fill="none" /></svg>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
