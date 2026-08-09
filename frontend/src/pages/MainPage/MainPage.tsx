import { observer } from 'mobx-react-lite';
import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router';

import { Button } from '@/components/ui/button';
import { Field, FieldLabel, FieldTitle } from '@/components/ui/field';
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
} from '@/components/ui/input-group';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Spinner } from '@/components/ui/spinner';
import { ROUTES } from '@/configs/routes';
import {
  hasRoleProgress,
  isTrainingPassed,
  rootStore,
  type Gender,
  type Role,
  type RoleProgress,
} from '@/stores';

import ProfileInfo from './ProfileInfo';
import RoleSwitch from './RoleSwitch';

const GENDERS: { id: Gender; label: string }[] = [
  { id: 'male', label: 'Мужской' },
  { id: 'female', label: 'Женский' },
];

const getExistingUserAction = (
  role: Role,
  progress: RoleProgress,
): { label: string; path: string } => {
  if (!isTrainingPassed(progress)) {
    return {
      label:
        progress.training.currentStep === 0
          ? 'Начать обучение'
          : 'Продолжить обучение',
      path: ROUTES.training.create(role),
    };
  }

  if (!progress.isExamPassed) {
    return { label: 'Сдать экзамен', path: ROUTES.exam.create(role) };
  }

  return { label: 'Посмотреть результаты', path: ROUTES.results.create(role) };
};

const MainPage = observer(() => {
  const { user: userStore } = rootStore;
  const navigate = useNavigate();

  const [role, setRole] = useState<Role>(userStore.role);
  const [name, setName] = useState(userStore.name);
  const [age, setAge] = useState(userStore.age);
  const [gender, setGender] = useState<Gender | null>(userStore.gender);

  useEffect(() => {
    if (!userStore.exists) {
      return;
    }

    setName(userStore.name);
    setAge(userStore.age);
    setGender(userStore.gender);
  }, [userStore.exists, userStore.name, userStore.age, userStore.gender]);

  const isReady = name.trim() !== '' && age !== '' && gender !== null;
  const progress = userStore.getProgress(role);
  const existingAction = getExistingUserAction(role, progress);
  const showReset = userStore.exists && hasRoleProgress(progress);
  const primaryLabel = userStore.exists
    ? existingAction.label
    : 'Перейти к обучению';
  const isBusy =
    userStore.submitStage.isLoading || userStore.resetStage.isLoading;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (userStore.exists) {
      userStore.setRole(role);
      void navigate(existingAction.path);

      return;
    }

    if (!gender) {
      return;
    }

    void userStore.createUser({ name, age, gender }).then(() => {
      userStore.setRole(role);
      void navigate(ROUTES.training.create(role));
    });
  };

  const handleReset = () => {
    void userStore.resetProgress(role).then(() => {
      rootStore.training.reset();
      rootStore.exam.reset();
      rootStore.results.reset();
    });
  };

  return (
    <div className="flex flex-col gap-10 py-8">
      <section className="flex flex-col items-center gap-4">
        <h1 className="text-center text-3xl font-extrabold tracking-tight text-balance sm:text-4xl">
          Научись распознавать мошенников вместе с{' '}
          <span className="rounded-xl bg-primary px-2 text-primary-foreground">
            Безопашей
          </span>
          !
        </h1>
        <p className="max-w-lg text-center text-muted-foreground text-balance">
          Пара шагов — и мы подберём сценарии под тебя.
        </p>
      </section>

      <section className="rounded-2xl border bg-card p-6 sm:p-8">
        <RoleSwitch value={role} onChange={setRole} />
      </section>

      {userStore.exists && userStore.gender ? (
        <ProfileInfo
          name={userStore.name}
          age={userStore.age}
          gender={userStore.gender}
        />
      ) : (
        <form
          id="profile-form"
          onSubmit={handleSubmit}
          className="flex flex-col gap-6 rounded-2xl border bg-card p-6 sm:p-8"
        >
          <h2 className="text-xl font-bold">Расскажи о себе</h2>

          <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
            <InputGroup className="sm:flex-1">
              <InputGroupInput
                id="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Как тебя зовут?"
                autoComplete="given-name"
              />
              <InputGroupAddon align="block-start">
                <InputGroupText>Имя</InputGroupText>
              </InputGroupAddon>
            </InputGroup>

            <InputGroup className="sm:w-32 sm:shrink-0">
              <InputGroupInput
                id="age"
                value={age}
                onChange={(event) =>
                  setAge(event.target.value.replace(/\D/g, '').slice(0, 3))
                }
                inputMode="numeric"
                maxLength={3}
                placeholder="18"
              />
              <InputGroupAddon align="block-start">
                <InputGroupText>Возраст</InputGroupText>
              </InputGroupAddon>
            </InputGroup>
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium text-muted-foreground">
              Пол
            </span>
            <RadioGroup
              value={gender}
              onValueChange={(value) => setGender(value as Gender)}
              className="grid-cols-2 gap-3 sm:max-w-sm"
            >
              {GENDERS.map(({ id, label }) => (
                <FieldLabel key={id} htmlFor={`gender-${id}`}>
                  <Field orientation="horizontal">
                    <FieldTitle>{label}</FieldTitle>
                    <RadioGroupItem value={id} id={`gender-${id}`} />
                  </Field>
                </FieldLabel>
              ))}
            </RadioGroup>
          </div>
        </form>
      )}

      <div className="flex flex-col gap-3">
        <Button
          type={userStore.exists ? 'button' : 'submit'}
          form={userStore.exists ? undefined : 'profile-form'}
          size="lg"
          disabled={
            isBusy ||
            (!userStore.exists && (!isReady || userStore.submitStage.isLoading))
          }
          className="w-full"
          onClick={
            userStore.exists
              ? () => {
                  userStore.setRole(role);
                  void navigate(existingAction.path);
                }
              : undefined
          }
        >
          {userStore.submitStage.isLoading ? (
            <Spinner className="size-4" />
          ) : null}
          {primaryLabel}
        </Button>

        {showReset ? (
          <Button
            type="button"
            variant="outline"
            size="lg"
            disabled={isBusy}
            className="w-full"
            onClick={handleReset}
          >
            {userStore.resetStage.isLoading ? (
              <Spinner className="size-4" />
            ) : null}
            Начать заново в роли {role === 'buyer' ? 'покупателя' : 'продавца'}
          </Button>
        ) : null}
      </div>
    </div>
  );
});

export default MainPage;
