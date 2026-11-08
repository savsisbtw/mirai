import { render } from 'preact';
import '@fontsource/dela-gothic-one/latin-400.css';
import '@fontsource/dela-gothic-one/cyrillic-400.css';
import '@fontsource/golos-text/latin-400.css';
import '@fontsource/golos-text/cyrillic-400.css';
import '@fontsource/golos-text/latin-500.css';
import '@fontsource/golos-text/cyrillic-500.css';
import '@fontsource/golos-text/latin-700.css';
import '@fontsource/golos-text/cyrillic-700.css';
import '@fontsource/jetbrains-mono/latin-400.css';
import '@fontsource/jetbrains-mono/cyrillic-400.css';
import '@fontsource/jetbrains-mono/latin-700.css';
import '@fontsource/jetbrains-mono/cyrillic-700.css';
import './styles.css';
import { App } from './App';

render(<App />, document.getElementById('app')!);
