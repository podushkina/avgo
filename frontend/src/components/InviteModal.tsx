import { useAppStore } from '../store/useAppStore';

export default function InviteModal() {
  const { inviteSeen, dismissInvite, startTraining } = useAppStore();

  if (inviteSeen) return null;

  return (
    <div className="modal__backdrop" role="dialog" aria-modal="true">
      <div className="modal">
        <span className="badge badge--info">Бесплатно, 5 минут</span>
        <h2 className="h1" style={{ marginTop: 14, fontSize: 24 }}>
          Проверьте, узнаете ли вы мошенника
        </h2>
        <p className="muted" style={{ marginTop: 10 }}>
          Разберём реальные ситуации из переписок покупателей и продавцов. Вы принимаете решения,
          сразу видите последствия и запоминаете признаки риска до того, как встретите их в живой
          сделке.
        </p>
        <div className="stack" style={{ marginTop: 22 }}>
          <button
            className="btn btn--primary btn--block"
            onClick={() => {
              dismissInvite();
              startTraining('seller');
            }}
          >
            Пройти тренажёр
          </button>
          <button className="btn btn--ghost btn--block" onClick={dismissInvite}>
            Позже
          </button>
        </div>
      </div>
    </div>
  );
}
