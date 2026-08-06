import { useQuery } from '@tanstack/react-query';
import { fetchProgress } from '../api/client';
import { useAppStore } from '../store/useAppStore';

export default function Home() {
  const { userId, startTraining, startDialog, go } = useAppStore();

  const { data: history } = useQuery({
    queryKey: ['progress', userId],
    queryFn: () => fetchProgress(userId!),
    enabled: Boolean(userId),
  });

  const best = history?.length ? Math.max(...history.map((h) => h.percent)) : null;

  return (
    <>
      <div className="card">
        <span className="badge badge--info">Тренажёр безопасных сделок</span>
        <h1 className="h1" style={{ marginTop: 14 }}>
          Научитесь распознавать мошенника до того, как отдадите деньги
        </h1>
        <p className="muted" style={{ marginTop: 12, maxWidth: 640 }}>
          Шесть ситуаций из настоящих переписок для каждой роли. Вы выбираете действие, сразу
          видите, чем оно закончилось бы, и получаете разбор признаков риска. После теста можно
          проверить себя в живом диалоге с ИИ, который играет мошенника.
        </p>

        {best !== null && (
          <p className="muted" style={{ marginTop: 16 }}>
            Ваш лучший результат: <strong>{best}%</strong>.{' '}
            <button className="navlink" style={{ padding: 0 }} onClick={() => go('progress')}>
              Посмотреть прогресс
            </button>
          </p>
        )}
      </div>

      <div className="card">
        <h2 className="h2">Выберите роль</h2>
        <p className="muted" style={{ marginTop: 6, marginBottom: 18 }}>
          Мошенники по-разному работают с теми, кто продаёт, и с теми, кто покупает.
        </p>

        <div className="rolegrid">
          <button className="rolecard" onClick={() => startTraining('seller')}>
            <div className="rolecard__emoji">📦</div>
            <div className="rolecard__title">Я продаю</div>
            <p className="muted">
              Фальшивые ссылки на «получение оплаты», коды из СМС, скриншоты переводов и схемы с
              переплатой.
            </p>
          </button>

          <button className="rolecard" onClick={() => startTraining('buyer')}>
            <div className="rolecard__emoji">🛍️</div>
            <div className="rolecard__title">Я покупаю</div>
            <p className="muted">
              Предоплата за «бронь», отказ от безопасной сделки, поддельные страницы оплаты и
              доплаты после платежа.
            </p>
          </button>
        </div>
      </div>

      <div className="card">
        <h2 className="h2">Живой диалог с мошенником</h2>
        <p className="muted" style={{ marginTop: 8, marginBottom: 18 }}>
          Собеседника отыгрывает языковая модель: она подстраивается под вашу роль и импровизирует,
          поэтому заранее заготовленного правильного ответа здесь нет. После разговора вы получите
          разбор применённых приёмов.
        </p>
        <div className="row">
          <button className="btn btn--primary" onClick={() => startDialog('seller', 'medium')}>
            Попробовать за продавца
          </button>
          <button className="btn btn--ghost" onClick={() => startDialog('buyer', 'medium')}>
            Попробовать за покупателя
          </button>
        </div>
      </div>
    </>
  );
}
