export type Title = {
  id: string;
  name: string;
  name_en: string;
  year: number;
  type: string;
  status: 'ongoing' | 'released' | 'announce';
  season: string;
  age: string;
  rating: number;
  poster: string;
  episodes: number | null;
  genres: string[];
  sources: string[];
};

export type TitleFull = Title & {
  description: string;
  backdrop: string;
};

export type Range = { start: number; stop: number };

export type Episode = {
  n: string;
  title?: string;
  hls?: Record<string, string>;
  opening?: Range;
  ending?: Range;
  duration?: number;
  url?: string;
};

export type Voice = {
  id: string;
  label: string;
  source: string;
  kind: 'hls' | 'embed';
  player: string;
  episodes: Episode[];
};

export type TitleDetail = TitleFull & {
  name_alt: string[];
  voices: Voice[];
};

export type HomeData = {
  featured: TitleFull[];
  latest: Title[];
  ongoing: Title[];
  popular: Title[];
};

export type CatalogData = {
  items: Title[];
  total: number;
  page: number;
  pages: number;
};

export type FiltersData = {
  genres: { name: string; count: number }[];
  years: number[];
  types: string[];
};

export type ScheduleDay = {
  day: number;
  name: string;
  items: { title: Title; episode: string }[];
};

export type StatusData = {
  ready: boolean;
  titles: number;
};

export type Brief = {
  id: string;
  name: string;
  poster: string;
  year: number;
  type: string;
};

export type HistoryItem = Brief & {
  ep: string;
  voice: string;
  pos: number;
  dur: number;
  at: number;
};

export type Settings = {
  server: string;
  autoSkip: boolean;
  quality: string;
};
