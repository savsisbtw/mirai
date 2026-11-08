import { useEffect, useState } from 'preact/hooks';

export type Route = {
  name: string;
  id: string;
  query: URLSearchParams;
};

export function parseHash(hash: string): Route {
  const [path, query = ''] = hash.replace(/^#/, '').split('?');
  const parts = path.split('/').filter(Boolean);
  return { name: parts[0] || 'home', id: decodeURIComponent(parts[1] || ''), query: new URLSearchParams(query) };
}

export function useHash(): string {
  const [hash, setHash] = useState(location.hash);
  useEffect(() => {
    const update = () => setHash(location.hash);
    addEventListener('hashchange', update);
    return () => removeEventListener('hashchange', update);
  }, []);
  return hash;
}
