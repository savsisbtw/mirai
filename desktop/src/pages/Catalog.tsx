import { useEffect, useLayoutEffect, useRef, useState } from 'preact/hooks';
import { cached, get, prefetch, useApi } from '../api';
import { Card, CardPlaceholders } from '../components/Card';
import { Empty, ErrorBox, PageHead } from '../components/Common';
import type { Route } from '../router';
import type { CatalogData, FiltersData, Title } from '../types';

const pageSize = 48;

type Filters = {
  q: string;
  genre: string;
  year: string;
  type: string;
  status: string;
  sort: string;
};

const emptyFilters: Filters = { q: '', genre: '', year: '', type: '', status: '', sort: 'popular' };

type Saved = {
  filters: Filters;
  pages: Title[][];
  total: number;
  pageCount: number;
  scroll: number;
};

const saved: Saved = { filters: emptyFilters, pages: [], total: 0, pageCount: 1, scroll: 0 };

const sortOptions = [
  ['popular', 'Популярные'],
  ['new', 'Новые'],
  ['rating', 'По рейтингу'],
  ['name', 'По названию']
];

const statusOptions = [
  ['ongoing', 'Онгоинг'],
  ['released', 'Вышло'],
  ['announce', 'Анонс']
];

function catalogPath(filters: Filters, page: number): string {
  const params = new URLSearchParams();
  (Object.keys(filters) as (keyof Filters)[]).forEach((key) => {
    if (filters[key]) params.set(key, filters[key]);
  });
  params.set('page', String(page));
  params.set('limit', String(pageSize));
  return '/v1/catalog?' + params.toString();
}

function filtersFromRoute(route: Route): Filters | null {
  const keys = Object.keys(emptyFilters) as (keyof Filters)[];
  if (!keys.some((key) => route.query.has(key))) return null;
  const next = { ...emptyFilters };
  keys.forEach((key) => {
    next[key] = route.query.get(key) || next[key];
  });
  return next;
}

function sameFilters(a: Filters, b: Filters): boolean {
  return (Object.keys(a) as (keyof Filters)[]).every((key) => a[key] === b[key]);
}

export function Catalog({ route }: { route: Route }) {
  const fromRoute = filtersFromRoute(route);
  if (fromRoute && !sameFilters(fromRoute, saved.filters)) {
    saved.filters = fromRoute;
    saved.pages = [];
    saved.total = 0;
    saved.pageCount = 1;
    saved.scroll = 0;
  }

  const [filters, setFilters] = useState<Filters>(saved.filters);
  const [queryText, setQueryText] = useState(saved.filters.q);
  const [pages, setPages] = useState<Title[][]>(saved.pages);
  const [total, setTotal] = useState(saved.total);
  const [pageCount, setPageCount] = useState(saved.pageCount);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const sentinel = useRef<HTMLDivElement>(null);
  const request = useRef(0);
  const filterData = useApi<FiltersData>('/v1/filters').data;

  const scroller = () => document.querySelector('main') as HTMLElement;

  useLayoutEffect(() => {
    scroller().scrollTop = saved.scroll;
    const node = scroller();
    return () => {
      saved.scroll = node.scrollTop;
    };
  }, []);

  useEffect(() => {
    saved.filters = filters;
    saved.pages = pages;
    saved.total = total;
    saved.pageCount = pageCount;
  }, [filters, pages, total, pageCount]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters((old) => (old.q === queryText ? old : { ...old, q: queryText }));
    }, 150);
    return () => clearTimeout(timer);
  }, [queryText]);

  const load = (page: number, replace: boolean) => {
    const id = ++request.current;
    setLoading(true);
    setError('');
    const path = catalogPath(filters, page);
    const ready = cached<CatalogData>(path);
    const task = ready ? Promise.resolve(ready) : get<CatalogData>(path);
    task
      .then((data) => {
        if (id !== request.current) return;
        setPages((old) => (replace ? [data.items] : [...old.slice(0, page - 1), data.items]));
        setTotal(data.total);
        setPageCount(data.pages);
        setLoading(false);
        if (replace) scroller().scrollTop = 0;
        if (page < data.pages) prefetch(catalogPath(filters, page + 1));
      })
      .catch((reason: Error) => {
        if (id !== request.current) return;
        setError(reason.message);
        setLoading(false);
      });
  };

  const firstRun = useRef(true);
  useEffect(() => {
    if (firstRun.current) {
      firstRun.current = false;
      if (pages.length > 0) return;
    }
    load(1, true);
  }, [filters]);

  useEffect(() => {
    const node = sentinel.current;
    if (!node || loading || error || pages.length === 0 || pages.length >= pageCount) return;
    const watcher = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) load(pages.length + 1, false);
      },
      { rootMargin: '1600px 0px' }
    );
    watcher.observe(node);
    return () => watcher.disconnect();
  }, [pages.length, pageCount, loading, error, filters]);

  const change = (key: keyof Filters, value: string) => {
    setFilters((old) => ({ ...old, [key]: value }));
  };

  const reset = () => {
    setQueryText('');
    setFilters(emptyFilters);
  };

  const dirty = !sameFilters(filters, emptyFilters) || queryText !== '';

  return (
    <div class="page">
      <PageHead kana="図鑑" title="Каталог">
        <span class="page-count">{total > 0 ? total + ' тайтлов' : ''}</span>
      </PageHead>
      <div class="filters">
        <input
          class="field field-search"
          type="search"
          placeholder="Поиск по названию"
          value={queryText}
          onInput={(event) => setQueryText((event.target as HTMLInputElement).value)}
        />
        <select class="field" value={filters.genre} onChange={(event) => change('genre', (event.target as HTMLSelectElement).value)}>
          <option value="">Все жанры</option>
          {filterData?.genres.map((genre) => (
            <option key={genre.name} value={genre.name}>{genre.name}</option>
          ))}
        </select>
        <select class="field" value={filters.year} onChange={(event) => change('year', (event.target as HTMLSelectElement).value)}>
          <option value="">Любой год</option>
          {filterData?.years.map((year) => (
            <option key={year} value={String(year)}>{year}</option>
          ))}
        </select>
        <select class="field" value={filters.type} onChange={(event) => change('type', (event.target as HTMLSelectElement).value)}>
          <option value="">Любой тип</option>
          {filterData?.types.map((type) => (
            <option key={type} value={type}>{type}</option>
          ))}
        </select>
        <select class="field" value={filters.status} onChange={(event) => change('status', (event.target as HTMLSelectElement).value)}>
          <option value="">Любой статус</option>
          {statusOptions.map(([value, label]) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </select>
        <select class="field" value={filters.sort} onChange={(event) => change('sort', (event.target as HTMLSelectElement).value)}>
          {sortOptions.map(([value, label]) => (
            <option key={value} value={value}>{label}</option>
          ))}
        </select>
        {dirty && (
          <button type="button" class="btn btn-small" onClick={reset}>Сбросить</button>
        )}
      </div>

      {pages.length === 0 && loading && <CardPlaceholders count={24} />}
      {pages.length === 0 && !loading && error && <ErrorBox text={error} retry={() => load(1, true)} />}
      {pages.length === 0 && !loading && !error && <Empty title="Ничего не найдено" text="Попробуйте изменить запрос или фильтры." />}

      <div class={loading && pages.length > 0 ? 'chunks dim' : 'chunks'}>
        {pages.map((items, index) => (
          <div class="grid chunk" key={index}>
            {items.map((item) => (
              <Card key={item.id} title={item} />
            ))}
          </div>
        ))}
      </div>

      {pages.length > 0 && error && <ErrorBox text={error} retry={() => load(pages.length + 1, false)} />}
      <div class="sentinel" ref={sentinel}>
        {pages.length > 0 && pages.length >= pageCount && <span>Это всё</span>}
      </div>
    </div>
  );
}
