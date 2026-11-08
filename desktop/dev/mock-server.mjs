import http from 'node:http';

const port = Number(process.env.PORT || 8787);

const firstWords = ['Клинок', 'Хроники', 'Последний', 'Тень', 'Небесный', 'Академия', 'Легенда', 'Пепел', 'Город', 'Дракон', 'Магия', 'Ночной', 'Стальной', 'Тихий', 'Девушка', 'Мир'];
const secondWords = ['рассвета', 'сумерек', 'героя', 'ветра', 'тысячи звёзд', 'забытых богов', 'лунного света', 'железа', 'другого мира', 'красного неба', 'бродячих псов', 'старой реки'];
const genreNames = ['Экшен', 'Сёнен', 'Сейнен', 'Комедия', 'Драма', 'Фэнтези', 'Романтика', 'Фантастика', 'Приключения', 'Детектив', 'Повседневность', 'Спорт', 'Ужасы'];
const types = ['ТВ', 'Фильм', 'ОВА', 'ONA', 'Спешл'];
const statuses = ['ongoing', 'released', 'announce'];
const seasons = ['Зима', 'Весна', 'Лето', 'Осень'];
const hues = [8, 28, 45, 160, 190, 210, 260, 330];

let seed = 7;
function random() {
  seed = (seed * 1664525 + 1013904223) % 4294967296;
  return seed / 4294967296;
}
function pick(list) {
  return list[Math.floor(random() * list.length)];
}

const titles = [];
for (let index = 0; index < 2400; index++) {
  const year = 1995 + Math.floor(random() * 32);
  const genres = new Set();
  const genreCount = 1 + Math.floor(random() * 3);
  while (genres.size < genreCount) genres.add(pick(genreNames));
  titles.push({
    id: 's' + (10000 + index),
    name: pick(firstWords) + ' ' + pick(secondWords) + (index % 7 === 0 ? ' ' + (2 + (index % 4)) : ''),
    name_en: 'Mock Title ' + index,
    year,
    type: random() < 0.7 ? 'ТВ' : pick(types),
    status: year >= 2026 ? pick(['ongoing', 'ongoing', 'announce']) : year === 2025 ? pick(statuses) : 'released',
    season: pick(seasons),
    age: pick(['12+', '16+', '18+']),
    rating: random() < 0.1 ? 0 : Math.round((5 + random() * 4.5) * 100) / 100,
    poster: '/v1/img?u=' + encodeURIComponent('mock://' + index),
    episodes: random() < 0.15 ? null : 6 + Math.floor(random() * 30),
    genres: [...genres],
    sources: random() < 0.5 ? ['anilibria', 'yummy'] : ['yummy'],
    views: Math.floor(random() * 100000),
    fresh: Math.floor(random() * 1000000),
    description:
      'Герои отправляются в путь через разрушенные земли, где каждая встреча меняет их судьбу. Старые долги всплывают на поверхность, а настоящий враг оказывается ближе, чем казалось. Мок-описание для проверки вёрстки и обрезки длинных текстов в карточках.'
  });
}

function brief(title) {
  const { views, fresh, description, ...rest } = title;
  void views;
  void fresh;
  void description;
  return rest;
}

function full(title) {
  return { ...brief(title), description: title.description, backdrop: title.poster };
}

function posterSvg(index) {
  const hue = hues[index % hues.length];
  const name = titles[index] ? titles[index].name : 'Mirai';
  const letter = name.slice(0, 1);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="600" viewBox="0 0 400 600">
<defs><pattern id="d" width="12" height="12" patternUnits="userSpaceOnUse"><circle cx="3" cy="3" r="2" fill="hsla(${hue},30%,90%,0.18)"/></pattern></defs>
<rect width="400" height="600" fill="hsl(${hue},38%,22%)"/>
<rect y="330" width="400" height="270" fill="hsl(${hue},45%,12%)"/>
<rect width="400" height="600" fill="url(#d)"/>
<path d="M0 600L260 220L400 380V600z" fill="hsl(${hue},40%,30%)"/>
<text x="40" y="250" font-family="Arial Black,sans-serif" font-size="220" fill="hsl(${hue},60%,85%)" opacity="0.9">${letter}</text>
</svg>`;
}

function sendJson(response, body, status = 200) {
  response.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', 'Access-Control-Allow-Origin': '*' });
  response.end(JSON.stringify(body));
}

function normalize(text) {
  return text.toLowerCase().replace(/ё/g, 'е');
}

function catalog(query) {
  let list = titles;
  const q = normalize(query.get('q') || '');
  if (q) {
    list = list
      .filter((title) => normalize(title.name + ' ' + title.name_en).includes(q))
      .sort((a, b) => Number(normalize(b.name).startsWith(q)) - Number(normalize(a.name).startsWith(q)));
  }
  const genre = query.get('genre');
  if (genre) {
    const wanted = genre.split(',');
    list = list.filter((title) => wanted.every((name) => title.genres.includes(name)));
  }
  const year = query.get('year');
  if (year) {
    const [from, to] = year.split('-').map(Number);
    list = list.filter((title) => title.year >= from && title.year <= (to || from));
  }
  if (query.get('type')) list = list.filter((title) => title.type === query.get('type'));
  if (query.get('status')) list = list.filter((title) => title.status === query.get('status'));
  const sort = query.get('sort') || 'popular';
  if (!q || sort !== 'popular') {
    list = [...list];
    if (sort === 'popular') list.sort((a, b) => b.views - a.views);
    if (sort === 'new') list.sort((a, b) => b.fresh - a.fresh);
    if (sort === 'rating') list.sort((a, b) => b.rating - a.rating);
    if (sort === 'name') list.sort((a, b) => a.name.localeCompare(b.name, 'ru'));
  }
  const limit = Math.min(100, Number(query.get('limit')) || 48);
  const page = Math.max(1, Number(query.get('page')) || 1);
  return {
    items: list.slice((page - 1) * limit, page * limit).map(brief),
    total: list.length,
    page,
    pages: Math.max(1, Math.ceil(list.length / limit))
  };
}

function home() {
  const byViews = [...titles].sort((a, b) => b.views - a.views);
  const byFresh = [...titles].sort((a, b) => b.fresh - a.fresh);
  const ongoing = byViews.filter((title) => title.status === 'ongoing');
  return {
    featured: ongoing.slice(0, 5).map(full),
    latest: byFresh.slice(0, 20).map(brief),
    ongoing: ongoing.slice(5, 25).map(brief),
    popular: byViews.slice(0, 20).map(brief)
  };
}

function filters() {
  const counts = new Map();
  titles.forEach((title) => title.genres.forEach((name) => counts.set(name, (counts.get(name) || 0) + 1)));
  const years = [...new Set(titles.map((title) => title.year))].sort((a, b) => b - a);
  return {
    genres: [...counts].map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count),
    years,
    types
  };
}

function detail(id) {
  const title = titles.find((item) => item.id === id);
  if (!title) return null;
  const count = title.episodes || 12;
  const hlsEpisodes = Array.from({ length: count }, (_, index) => {
    const n = String(index + 1);
    const base = 'https://example.invalid/' + id + '/' + n;
    return {
      n,
      title: 'Серия ' + n,
      hls: { 480: base + '/480.m3u8', 720: base + '/720.m3u8', 1080: base + '/1080.m3u8' },
      opening: { start: 12, stop: 101 },
      ending: { start: 1300, stop: 1390 },
      duration: 1420
    };
  });
  const embed = (voiceId, label, player, total) => ({
    id: voiceId,
    label,
    source: 'yummy',
    kind: 'embed',
    player,
    episodes: Array.from({ length: total }, (_, index) => ({ n: String(index + 1), url: 'https://kodikplayer.com/seria/' + id + '/' + (index + 1) }))
  });
  const voices = [];
  if (title.sources.includes('anilibria')) {
    voices.push({ id: 'al', label: 'AniLibria', source: 'anilibria', kind: 'hls', player: '', episodes: hlsEpisodes });
  }
  voices.push(embed('ym-anidub-kodik', 'AniDub', 'Kodik', count));
  voices.push(embed('ym-sovet-aksor', 'SovetRomantica', 'Aksor', Math.max(1, count - 2)));
  return { ...full(title), name_alt: ['Альтернативное название', 'Alt title ' + id], voices };
}

function schedule() {
  const names = ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье'];
  const ongoing = titles.filter((title) => title.status === 'ongoing');
  return names.map((name, index) => ({
    day: index + 1,
    name,
    items: ongoing.slice(index * 5, index * 5 + 5 + (index % 3)).map((title, position) => ({ title: brief(title), episode: String(3 + position) }))
  }));
}

const startedAt = Date.now();
const buildDelayMs = Number(process.env.BUILD_DELAY || 0);

const server = http.createServer((request, response) => {
  const url = new URL(request.url, 'http://localhost');
  const path = url.pathname;
  if (request.method === 'OPTIONS') {
    response.writeHead(204, { 'Access-Control-Allow-Origin': '*', 'Access-Control-Allow-Headers': '*' });
    response.end();
    return;
  }
  if (path === '/v1/status') {
    const ready = Date.now() - startedAt >= buildDelayMs;
    return sendJson(response, { ready, titles: titles.length, anilibria: 'done', yummy: 'done', version: 'mock' });
  }
  if (path === '/v1/home') return sendJson(response, home());
  if (path === '/v1/catalog') return sendJson(response, catalog(url.searchParams));
  if (path === '/v1/filters') return sendJson(response, filters());
  if (path === '/v1/schedule') return sendJson(response, schedule());
  if (path.startsWith('/v1/title/')) {
    const body = detail(decodeURIComponent(path.slice('/v1/title/'.length)));
    return body ? sendJson(response, body) : sendJson(response, { error: 'тайтл не найден' }, 404);
  }
  if (path === '/v1/img') {
    const index = Number((url.searchParams.get('u') || '').replace('mock://', '')) || 0;
    response.writeHead(200, { 'Content-Type': 'image/svg+xml', 'Cache-Control': 'public, max-age=31536000, immutable', 'Access-Control-Allow-Origin': '*' });
    response.end(posterSvg(index));
    return;
  }
  if (path === '/v1/embed') {
    response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8', 'Access-Control-Allow-Origin': '*' });
    response.end('<body style="margin:0;background:#111;color:#eee;font:20px sans-serif;display:grid;place-items:center;height:100vh">Заглушка плеера<br>' + (url.searchParams.get('u') || '') + '</body>');
    return;
  }
  sendJson(response, { error: 'не найдено' }, 404);
});

server.listen(port, '127.0.0.1', () => {
  console.log('mock server http://127.0.0.1:' + port + ', titles: ' + titles.length);
});
