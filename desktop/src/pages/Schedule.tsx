import { useState } from 'preact/hooks';
import { useApi } from '../api';
import { Card } from '../components/Card';
import { Empty, ErrorBox, PageHead } from '../components/Common';
import type { ScheduleDay } from '../types';

const shortNames = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

export function Schedule() {
  const { data, error, retry } = useApi<ScheduleDay[]>('/v1/schedule');
  const today = ((new Date().getDay() + 6) % 7) + 1;
  const [picked, setPicked] = useState(today);

  if (!data) {
    if (error) return <ErrorBox text={error} retry={retry} />;
    return <div class="page"><PageHead kana="予定" title="Расписание" /></div>;
  }

  const current = data.find((item) => item.day === picked);

  return (
    <div class="page">
      <PageHead kana="予定" title="Расписание" />
      <div class="days">
        {data.map((day) => (
          <button
            type="button"
            key={day.day}
            class={day.day === picked ? 'day on' : 'day'}
            onClick={() => setPicked(day.day)}
          >
            <b>{shortNames[day.day - 1] || day.name}</b>
            <span>{day.items.length}</span>
            {day.day === today && <i>сегодня</i>}
          </button>
        ))}
      </div>
      {current && current.items.length > 0 ? (
        <div class="grid">
          {current.items.map((item) => (
            <Card key={item.title.id + item.episode} title={item.title} badge={'Серия ' + item.episode} />
          ))}
        </div>
      ) : (
        <Empty title="В этот день выходов нет" />
      )}
    </div>
  );
}
