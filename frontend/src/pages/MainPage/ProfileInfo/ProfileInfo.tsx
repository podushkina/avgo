import type { Gender } from '@/stores';

const GENDER_LABELS: Record<Gender, string> = {
  male: 'Мужской',
  female: 'Женский',
};

type ProfileInfoProps = {
  name: string;
  age: string;
  gender: Gender;
};

const ProfileInfo = ({ name, age, gender }: ProfileInfoProps) => {
  const rows = [
    { label: 'Имя', value: name },
    { label: 'Возраст', value: age },
    { label: 'Пол', value: GENDER_LABELS[gender] },
  ];

  return (
    <div className="flex flex-col gap-6 rounded-2xl border bg-card p-6 sm:p-8">
      <h2 className="text-xl font-bold">О тебе</h2>

      <dl className="grid gap-4 sm:grid-cols-3">
        {rows.map(({ label, value }) => (
          <div
            key={label}
            className="flex flex-col gap-1 rounded-xl border bg-muted/40 px-4 py-3"
          >
            <dt className="text-sm font-medium text-muted-foreground">
              {label}
            </dt>
            <dd className="text-lg font-semibold tracking-tight">{value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
};

export default ProfileInfo;
