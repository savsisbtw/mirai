import { Component, h } from 'preact';
import type { ComponentType } from 'preact';

export function memo<P extends object>(Inner: ComponentType<P>): ComponentType<P> {
  class Memo extends Component<P> {
    shouldComponentUpdate(next: P) {
      const keys = Object.keys(next) as (keyof P)[];
      if (keys.length !== Object.keys(this.props).length) return true;
      return keys.some((key) => next[key] !== this.props[key]);
    }

    render() {
      return h(Inner as ComponentType<object>, this.props);
    }
  }
  return Memo as unknown as ComponentType<P>;
}
