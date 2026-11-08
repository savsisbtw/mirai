# Mirai - контракт API (server <-> клиенты)

Сервер: Go, только стандартная библиотека. Слушает `127.0.0.1:8787` (флаги `-addr`, `-data`).
Все ответы JSON, UTF-8, gzip, CORS `*`. Ошибка: HTTP 4xx/5xx + `{"error":"текст"}`.
Любой клиент (десктоп, потом iOS/Android) ходит ТОЛЬКО сюда, в AniLiberty/Yummy напрямую не ходит.

## Идея
Один каталог "все аниме сразу". Сервер при старте (и раз в 6 ч) обходит AniLiberty (~1800 тайтлов,
`POST https://aniliberty.top/api/v1/anime/catalog/releases` limit 50) и Yummy (`https://api.yani.tv/anime?limit=100&offset=N`,
offset максимум 20000, без токена GET работает) и склеивает дубли по id: shikimori_id, myanimelist_id,
anilibria_alias (Yummy `remote_ids`). Снимок лежит на диске (`<data>/catalog.json`), при запуске грузится сразу,
обход идёт в фоне. Поиск и фильтры целиком в памяти (быстро, без запросов наружу).
Озвучки берутся живьём при открытии тайтла (кэш 10 мин): AniLiberty (HLS, свой плеер) и Yummy (iframe: Kodik, Alloha, Aksor, CVH/VK).

## Title (карточка)
```json
{
  "id": "s61999",
  "name": "Маска вермильона",
  "name_en": "Shuiro no Kamen",
  "year": 2026,
  "type": "ТВ",
  "status": "ongoing",
  "season": "Осень",
  "age": "16+",
  "rating": 7.66,
  "poster": "/v1/img?u=<urlencoded>",
  "episodes": 12,
  "genres": ["Сейнен", "Экшен"],
  "sources": ["anilibria", "yummy"]
}
```
- `id`: `s<shikimori>` если есть, иначе `a<anilibria_alias>`, иначе `y<yummy_id>`. Стабильный, URL-safe.
- `status`: `ongoing` | `released` | `announce`. `rating` 0-10 или 0. `episodes` число или null.
- `poster` всегда относительный путь на этом же сервере (клиент добавляет базовый URL).
- `sources` - где реально есть видео: `anilibria`, `yummy`.

## Эндпоинты
`GET /v1/status` -> `{"ready":true,"titles":19800,"anilibria":"done|running|error","yummy":"done|running|error","version":"0.1.0"}`
`ready` = в памяти есть хоть что-то.

`GET /v1/home` -> 
```json
{"featured":[TitleFull x5],"latest":[Title x20],"ongoing":[Title x20],"popular":[Title x20]}
```
`featured` - ongoing с лучшими рейтингом/популярностью и нормальным постером; `TitleFull` = Title + `description` + `backdrop` (тот же poster, больший размер если есть).
`latest` - по свежести (AniLiberty `fresh_at`), `popular` - по `added_in_users_favorites`/Yummy views.

`GET /v1/catalog?q=&genre=&year=&type=&status=&sort=&page=1&limit=48`
-> `{"items":[Title],"total":1234,"page":1,"pages":26}`
- `q`: поиск без регистра по name/name_en/alt, с ё=е, подстрока; ранжирование: точное начало слова > подстрока.
- `genre`: имя жанра (можно несколько через запятую = И). `year`: `2020` или `2018-2022`. `type`, `status` - значения как в Title.
- `sort`: `popular` (по умолчанию) | `new` | `rating` | `name`. `limit` макс 100.

`GET /v1/filters` -> `{"genres":[{"name":"Экшен","count":900}],"years":[2026,...],"types":["ТВ","Фильм",...]}`

`GET /v1/title/{id}` -> TitleFull + 
```json
{
  "name_alt": ["..."],
  "voices": [
    {"id":"al","label":"AniLibria","source":"anilibria","kind":"hls","player":"","episodes":[
      {"n":"1","title":"Название серии","hls":{"480":"https://...m3u8","720":"...","1080":"..."},
       "opening":{"start":12,"stop":101},"ending":{"start":1300,"stop":1390},"duration":1420}]},
    {"id":"ym-anidub-kodik","label":"AniDub","source":"yummy","kind":"embed","player":"Kodik","episodes":[
      {"n":"1","url":"https://kodikplayer.com/..."}]}
  ]
}
```
- AniLiberty: серии через `GET https://aniliberty.top/api/v1/anime/releases/{id|alias}` (поле `episodes`), `hls_480/720/1080`, `opening/ending {start,stop}`.
- Yummy: `GET https://api.yani.tv/anime/{yummy_id}/videos` -> группируем по (dubbing без "Озвучка ", player без "Плеер "), в каждой группе серии по `number`. Берём только https-плееры с хостов kodikplayer.com, yani.tv, yummyani.me, aksor.tv (включая поддомены). Нет в AniLiberty - будут только yummy-озвучки.
- Порядок voices: AniLibria первой, дальше Yummy по числу серий убыв.
- Если внешний источник лёг - отдаём что есть, без 500.

`GET /v1/schedule` -> `[{"day":1,"name":"Понедельник","items":[{"title":Title,"episode":"5"}]}]` (из AniLiberty `/anime/schedule/week`, сопоставление с каталогом по anilibria_alias, нет в каталоге - пропуск).

`GET /v1/img?u=<url>` - прокси картинок, allowlist хостов `aniliberty.top`, `*.yani.tv`; кэш на диске `<data>/img/`, заголовок `Cache-Control: public, max-age=31536000, immutable`. Принимает `//host/...` и относительные `/storage/...` (считаем что aniliberty.top).

`GET /v1/embed?u=<player url>` -> HTML страница с одним fullscreen iframe на этот url (`allow="autoplay; fullscreen"`, `referrerpolicy="origin"`), чтобы у плеера был http(s) Referer. Только хосты из списка выше.

## Быстродействие (жёсткое требование)
- Запросы клиента из памяти: p99 < 5 мс на `/v1/catalog` и `/v1/home` на 20k тайтлов. Без блокировок на горячем пути (атомарная подмена снимка).
- Предвычисленные индексы: lower-case строки для поиска, отсортированные срезы под каждый `sort`.
- gzip, `ETag`/`304` на `/v1/home`, `/v1/filters`, `/v1/catalog`.
- Обход источников: параллельно (AniLiberty и Yummy одновременно, Yummy 6 воркеров), ретраи с бэкоффом, таймаут 10 с, сервер не падает, если источник недоступен.
- Старт < 300 мс до готовности отвечать (из снимка на диске).
