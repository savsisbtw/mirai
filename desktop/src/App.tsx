import { useEffect, useLayoutEffect, useRef, useState } from 'preact/hooks';
import { fetchJson, serverUrl } from './api';
import { Logo, TitleBar } from './components/TitleBar';
import { Win10Notice } from './components/Win10Notice';
import { Catalog } from './pages/Catalog';
import { Home } from './pages/Home';
import { Favorites, History } from './pages/Library';
import { Schedule } from './pages/Schedule';
import { Settings } from './pages/Settings';
import { TitlePage } from './pages/TitlePage';
import { parseHash, useHash } from './router';
import type { StatusData } from './types';

const navItems = [
  ['home', 'Главная'],
  ['catalog', 'Каталог'],
  ['schedule', 'Расписание'],
  ['favorites', 'Избранное'],
  ['history', 'История'],
  ['settings', 'Настройки']
];

type Gate = 'check' | 'ready' | 'building' | 'down';

function useGate(): [Gate, () => void] {
  const [gate, setGate] = useState<Gate>('check');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let alive = true;
    let timer = 0;
    const check = () => {
      fetchJson<StatusData>('/v1/status')
        .then((status) => {
          if (!alive) return;
          setGate(status.ready ? 'ready' : 'building');
          if (!status.ready) timer = window.setTimeout(check, 2500);
        })
        .catch(() => {
          if (!alive) return;
          setGate('down');
          timer = window.setTimeout(check, 2500);
        });
    };
    check();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [attempt]);

  return [gate, () => setAttempt((value) => value + 1)];
}

export function App() {
  const hash = useHash();
  const route = parseHash(hash);
  const [gate, recheck] = useGate();
  const main = useRef<HTMLElement>(null);

  useLayoutEffect(() => {
    if (main.current && route.name !== 'catalog') main.current.scrollTop = 0;
  }, [route.name, route.id]);

  let page;
  if (gate === 'check') page = null;
  else if (gate === 'building') {
    page = (
      <div class="notice notice-wide">
        <p class="notice-title">Каталог собирается</p>
        <p class="notice-text">Сервер только что запустился и собирает список тайтлов. Это займёт немного времени, страница обновится сама.</p>
      </div>
    );
  } else if (gate === 'down' && route.name !== 'settings') {
    page = (
      <div class="notice notice-wide">
        <p class="notice-title">Сервер не отвечает</p>
        <p class="notice-text">Адрес: {serverUrl()}. Если приложение только что запустилось, подождите пару секунд, попытки идут сами.</p>
        <div class="notice-actions">
          <button type="button" class="btn" onClick={recheck}>Повторить</button>
          <a class="btn" href="#/settings">Настройки</a>
        </div>
      </div>
    );
  } else if (route.name === 'catalog') page = <Catalog route={route} />;
  else if (route.name === 'title') page = <TitlePage route={route} key={route.id} />;
  else if (route.name === 'schedule') page = <Schedule />;
  else if (route.name === 'favorites') page = <Favorites />;
  else if (route.name === 'history') page = <History />;
  else if (route.name === 'settings') page = <Settings />;
  else page = <Home />;

  const activeName = route.name === 'title' ? 'catalog' : route.name;

  return (
    <div class="app">
      <TitleBar />
      <div class="shell">
        <nav class="rail" aria-label="Разделы">
          <a class="rail-logo" href="#/">
            <Logo size={28} />
          </a>
          {navItems.map(([name, label], index) => (
            <a key={name} class={activeName === name ? 'rail-link on' : 'rail-link'} href={'#/' + (name === 'home' ? '' : name)}>
              <span>{String(index + 1).padStart(2, '0')}</span>
              {label}
            </a>
          ))}
          <span class="rail-kana" aria-hidden="true">アニメ</span>
        </nav>
        <main ref={main}>{page}</main>
      </div>
      <Win10Notice />
    </div>
  );
}
