# Backend API

Контракт ручек, которые ожидает фронт. Пользователь анонимный: идентификация через `httpOnly`-куку (фронт куку не читает и не шлёт в body).

Общие типы:

```ts
type Role = 'buyer' | 'seller';
type Gender = 'male' | 'female';
type ExamVerdict = 'passed' | 'failed';

type RoleProgress = {
  training: {
    currentStep: number;
    totalSteps: number;
  };
  isExamPassed: boolean;
};
```

`training.currentStep`:

- `0` — обучение не начато; пользователь на первом шаге
- `1 … totalSteps - 1` — обучение в процессе
- `totalSteps` — обучение пройдено

---

## Пользователь

### `GET /me`

Кто вызывает: при создании `UserStore` (любая страница приложения).

**Request:** без body, только cookie.

**Response:**

```ts
type MeResponse = {
  exists: boolean;
  user: {
    name: string;
    age: string;
    gender: Gender;
    buyer: RoleProgress;
    seller: RoleProgress;
  } | null;
};
```

- `exists: false` → `user: null` (новый аноним, формы профиля ещё нет).
- `exists: true` → заполненный `user`, прогресс по обеим ролям.

---

### `POST /users`

Создание профиля при первом прохождении.

**Request:**

```ts
type CreateUserRequest = {
  name: string;
  age: string;
  gender: Gender;
};
```

**Response:**

```ts
type CreateUserResponse = {
  user: {
    name: string;
    age: string;
    gender: Gender;
    buyer: RoleProgress; // currentStep: 0, isExamPassed: false
    seller: RoleProgress;
  };
};
```

После успеха бэк ставит `httpOnly`-куку сессии.

---

### `POST /progress/reset`

Сброс прогресса обучения и экзамена для выбранной роли. Сам пользователь не удаляется.

**Request:**

```ts
type ResetProgressRequest = {
  role: Role;
};
```

**Response:**

```ts
type ResetProgressResponse = RoleProgress; // currentStep: 0, isExamPassed: false
```

---

## Обучение

### `GET /training/current-step?role={role}`

Текущий шаг обучения для роли. Шаг в URL фронта не хранится — всегда запрашивается с бэка.

**Query:** `role: Role`

**Response:**

```ts
type TrainingStepResponse = {
  currentStep: number; // 1-based
  totalSteps: number;
  productName: string;
  message: string; // реплика Безопаши в чате
  variants: { id: number; text: string }[];
};
```

---

### `POST /training/answer`

Ответ пользователя на шаг.

**Request:**

```ts
type SubmitAnswerRequest = {
  role: Role;
  answer_id: number;
};
```

**Response:**

```ts
type SubmitAnswerResponse = {
  isCorrect: boolean;
  explanation: string;
};
```

---

## Экзамен

### `GET /exam/start?role={role}`

Старт диалога с ИИ: первое сообщение модели.

**Query:** `role: Role`

**Response:**

```ts
type ExamStartResponse = {
  message: string; // первая реплика ассистента
};
```

---

### `POST /exam/message`

Сообщение пользователя → ответ ИИ-модели.

**Request:**

```ts
type ExamMessageRequest = {
  role: Role;
  text: string;
};
```

**Response:**

```ts
type ExamReplyResponse = {
  message: string; // ответ модели
  isFinished: boolean; // диалог закончен?
  verdict: ExamVerdict | null; // 'passed' | 'failed' | null
  explanation: string | null; // пояснение к вердикту; null пока диалог идёт
};
```

Правила для фронта:

| `isFinished` | `verdict`           | `explanation` | UI                                      |
| ------------ | ------------------- | ------------- | --------------------------------------- |
| `false`      | `null`              | `null`        | продолжаем чат, инпут активен           |
| `true`       | `'passed'`/`'failed'` | строка      | инпут скрыт, блок результата + кнопка к результатам |

Когда `isFinished === true`, бэк выставляет `isExamPassed` по роли согласно `verdict` (или отдельным полем прогресса — фронт на моках сам помечает прогресс при переходе к результатам).

---

## Результаты

### `POST /results`

Итоги прохождения для выбранной роли.

**Request:**

```ts
type ResultsRequest = {
  role: Role;
};
```

**Response:**

```ts
type ResultsResponse = {
  role: Role;
  training: {
    correctSteps: number;
    totalSteps: number;
  };
  exam: {
    verdict: ExamVerdict;
    explanation: string;
  };
  tips: string[];
};
```

---

## Сводка

| Method | Path                         | Зачем                                      |
| ------ | ---------------------------- | ------------------------------------------ |
| `GET`  | `/me`                        | Есть ли юзер + профиль + прогресс          |
| `POST` | `/users`                     | Создать пользователя                       |
| `POST` | `/progress/reset`            | Сбросить прогресс выбранной роли           |
| `GET`  | `/training/current-step`     | Текущий шаг обучения (`?role=`)            |
| `POST` | `/training/answer`           | Ответ на шаг обучения                      |
| `GET`  | `/exam/start`                | Старт экзамена, первое сообщение (`?role=`) |
| `POST` | `/exam/message`              | Сообщение в экзамене → ответ ИИ            |
| `POST` | `/results`                   | Результаты по роли                         |

Пути `/users` и `/progress/reset` на фронте пока только в комментариях к мокам; имена можно согласовать, схемы выше — то, что фронт реально ждёт по полям.
