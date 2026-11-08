import { useEffect, useState } from 'preact/hooks';
import { readFlag, writeFlag } from '../store';

const flagKey = 'mirai.win10.dismissed';

type PlatformData = { userAgentData?: { getHighEntropyValues: (hints: string[]) => Promise<{ platformVersion?: string }> } };

async function isWindows10(): Promise<boolean> {
  const data = (navigator as unknown as PlatformData).userAgentData;
  if (data) {
    try {
      const values = await data.getHighEntropyValues(['platformVersion']);
      const major = parseInt(values.platformVersion || '0', 10);
      if (major > 0) return major < 13;
    } catch {
      return false;
    }
  }
  return /Windows NT [1-6]\.\d/.test(navigator.userAgent) && !/Windows NT 10\.0/.test(navigator.userAgent);
}

export function Win10Notice() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (readFlag(flagKey)) return;
    isWindows10().then(setVisible);
  }, []);

  if (!visible) return null;

  const close = () => {
    writeFlag(flagKey, true);
    setVisible(false);
  };

  return (
    <div class="toast" role="status">
      <p>На Windows 10 могут быть баги, хз</p>
      <button type="button" aria-label="Закрыть" onClick={close}>
        <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1.4" fill="none" /></svg>
      </button>
    </div>
  );
}
