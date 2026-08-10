# Backend API

Контракт ручек, которые ожидает фронт. Пользователь анонимный: идентификация через `httpOnly`-куку (фронт куку не читает и не шлёт в body).

## Базовый путь

Все пути ниже указаны **относительно `/api`**. Алиас `/api/v1` оставлен для совместимости и работает так же.

```
GET /api/me         ✅ 200 application/json
GET /api/v1/me      ✅ 200 application/json   ← алиас, работает так же
GET /me             ✅ 404 application/json   ← честная ошибка, а не молчаливый HTML
```

Голых путей нет намеренно: они столкнулись бы со страничными маршрутами приложения. `/exam/start` неотличим от `/exam/:role`, где роль равна `start`, а `/training/current-step` — от `/training/:role`.

Запрос к любому неизвестному пути возвращает **404 с тем же конвертом ошибки**, что и остальные ручки:

```json
{ "error": { "code": "NOT_FOUND", "message": "Неизвестный путь. API доступен под /api, документация - /api/docs" } }
```

Исключение — навигация в браузере: запрос с `Accept: text/html` отдаётся приложению, чтобы работали клиентские маршруты и обновление страницы. То есть `fetch` и `curl` получают честный отказ, а адресная строка — страницу.

Интерактивная документация: **`/api/docs`** (Swagger UI), спецификация — `/api/openapi.yaml`.

## Общие типы

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

  // сверх контракта, можно игнорировать
  status?:
    | 'not_started'
    | 'training_in_progress'
    | 'training_passed'
    | 'exam_in_progress'
    | 'exam_passed'
    | 'exam_failed';
  isTrainingPassed?: boolean;
};
```

`training.currentStep`:

- `0` — обучение не начато; пользователь на первом шаге
- `1 … totalSteps - 1` — обучение в процессе
- `totalSteps` — обучение пройдено

По `status` можно решать, какую кнопку показать, не собирая состояние на клиенте.

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

- `exists: false` → `user: null` (новый аноним, формы профиля ещё нет). Статус **200**, не 404.
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

После успеха бэк ставит `httpOnly`-куку сессии. Повторный вызов с уже существующей кукой обновляет профиль и **не сбрасывает прогресс**.

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

Есть алиас с ролью в пути: `POST /progress/reset/{role}`, тело не нужно.

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

> ⚠️ Здесь `currentStep` — **1-based номер текущего шага**, а в `RoleProgress.training.currentStep` — число **уже завершённых** шагов. Поля называются одинаково, но означают разное.

Правильный ответ наружу не отдаётся: в `variants` только `id` и `text`. Проверка выполняется на сервере.

**Ошибки:** `409 TRAINING_ALREADY_PASSED`, если обучение по роли уже пройдено.

---

### `POST /training/answer`

Ответ пользователя на шаг.

**Request:**

```ts
type SubmitAnswerRequest = {
  role: Role;
  answer_id: number; // ⚠️ единственное snake_case-поле в контракте
  stepNumber?: number; // необязателен, см. ниже
};
```

`answerId` в camelCase принимается наравне с `answer_id`.

Если передать `stepNumber` и он не совпадёт с текущим шагом — ответ не запишется, указатель не сдвинется, придёт `409 STEP_MISMATCH` с актуальным состоянием в `details`. Так гасится двойной клик и повтор запроса. Без `stepNumber` шаг берётся из состояния на сервере.

**Response:**

```ts
type SubmitAnswerResponse = {
  isCorrect: boolean;
  explanation: string;

  // сверх контракта
  correctId?: number;
  currentStep?: number; // уже сдвинутый указатель
  totalSteps?: number;
  isTrainingFinished?: boolean;
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
  message: string; // последняя реплика собеседника

  // сверх контракта
  sessionId?: string;
  messages?: { id: number; author: 'scammer' | 'user'; text: string; createdAt: string }[];
  isFinished?: boolean;
  verdict?: ExamVerdict | null;
  explanation?: string | null;
  cycle?: number;
  maxCycles?: number;
};
```

Если активная сессия уже есть — возвращается **она** вместе со всей историей, новая не создаётся и модель не дёргается. Именно по `messages` чат восстанавливается после перезагрузки страницы.

Есть алиас `POST /exam/start` с ролью в теле.

**Ошибки:** `409 TRAINING_NOT_PASSED`, если обучение по роли не пройдено.

---

### `POST /exam/message`

Сообщение пользователя → ответ ИИ-модели.

**Request:**

```ts
type ExamMessageRequest = {
  role: Role;
  text: string; // до 1000 символов
};
```

**Response:**

```ts
type ExamReplyResponse = {
  message: string; // ответ модели
  isFinished: boolean; // диалог закончен?
  verdict: ExamVerdict | null;
  explanation: string | null; // пояснение к вердикту; null пока диалог идёт

  // сверх контракта
  cycle?: number; // сделано ходов
  maxCycles?: number; // страховочный потолок
};
```

Правила для фронта:

| `isFinished` | `verdict`             | `explanation` | UI                                                  |
| ------------ | --------------------- | ------------- | --------------------------------------------------- |
| `false`      | `null`                | `null`        | продолжаем чат, инпут активен                       |
| `true`       | `'passed'`/`'failed'` | строка        | инпут скрыт, блок результата + кнопка к результатам |

**Экзамен завершается по исходу разговора, а не по счётчику ходов:**

- пользователь выдал код из СМС, данные карты или согласился платить в обход площадки — `failed` немедленно;
- твёрдо отказался и дал понять, что разговор окончен — `passed`;
- собеседник перепробовал все свои приёмы — `passed`;
- достигнут `maxCycles` — страховка, срабатывает редко.

Поэтому `cycle` не стоит показывать как «осталось N вопросов»: разговор может кончиться раньше.

**Ошибки:** `400 MESSAGE_TOO_LONG`, `404 SESSION_NOT_FOUND`, `429 RATE_LIMITED` (20 сообщений в минуту на сессию), `503 LLM_UNAVAILABLE`.

---

### `POST /exam/finish`

Досрочно завершить разговор по кнопке.

**Request:** `{ role: Role }`

**Response:**

```ts
{
  verdict: ExamVerdict;
  explanation: string;
  isFinished: boolean;
  cycle?: number;
  maxCycles?: number;
}
```

Добровольный выход без критических ошибок засчитывается как успех.

> ⚠️ **Результат записывается только при завершении экзамена** — через `/exam/message` с `isFinished: true` или через `/exam/finish`. Без этого `GET /results` вернёт `404 RESULTS_NOT_READY`.

---

### `POST /exam/restart`

Начать экзамен заново: закрывает активную сессию и создаёт новую с чистой историей.

**Request:** `{ role: Role }`
**Response:** то же, что у `GET /exam/start`.

---

## Результаты

### `GET /results?role={role}`

Итоги прохождения для выбранной роли. Есть алиас `POST /results` с ролью в теле.

**Response:**

```ts
type ResultsResponse = {
  role: Role;
  training: {
    correctSteps: number;
    totalSteps: number;
    answers?: { stepNumber: number; isCorrect: boolean }[];
  };
  exam: {
    verdict: ExamVerdict;
    explanation: string;
    cyclesPassed?: number;
    criticalMistakes?: string[];
    endReason?:
      | 'critical_mistake'
      | 'refused_and_ended'
      | 'tactics_exhausted'
      | 'user_finished'
      | 'limit_reached';
  };
  tips: string[];

  // сверх контракта
  score?: number; // 0…100
  grade?: 'Новичок' | 'Осторожный' | 'Уверенный' | 'Эксперт';
  strengths?: string[];
  weaknesses?: string[];
};
```

`strengths` и `weaknesses` персональные — считаются по реальному диалогу, а не шаблонные.

Результат читается из хранилища и не пересчитывается: он записывается один раз в момент завершения экзамена.

**Ошибки:** `404 RESULTS_NOT_READY`, если экзамен ещё не завершён.

---

## Формат ошибок

Единый на все ручки:

```json
{ "error": { "code": "STEP_MISMATCH", "message": "Шаг не совпадает с текущим" } }
```

| Код                        | Статус | Когда                                          |
| -------------------------- | ------ | ---------------------------------------------- |
| `USER_NOT_FOUND`           | 401    | Нет куки или пользователь удалён                |
| `TRAINING_NOT_PASSED`      | 409    | Экзамен до завершения обучения                  |
| `TRAINING_ALREADY_PASSED`  | 409    | Запрос шага, когда обучение пройдено            |
| `STEP_MISMATCH`            | 409    | `stepNumber` не совпадает с текущим шагом       |
| `INVALID_OPTION`           | 400    | Вариант не относится к текущему шагу            |
| `SESSION_NOT_FOUND`        | 404    | Нет активной сессии экзамена                    |
| `SESSION_ALREADY_FINISHED` | 409    | Экзамен уже завершён                            |
| `MESSAGE_TOO_LONG`         | 400    | Сообщение длиннее 1000 символов                 |
| `RATE_LIMITED`             | 429    | Больше 20 сообщений в минуту на сессию          |
| `RESULTS_NOT_READY`        | 404    | Результатов ещё нет                             |
| `LLM_UNAVAILABLE`          | 503    | Модель недоступна                               |
| `NOT_FOUND`                | 404    | Неизвестный путь                                |

У `STEP_MISMATCH` в `error.details` приходит актуальный `RoleProgress` — им можно сразу поправить состояние на клиенте.

---

## Сводка

| Method       | Path                            | Зачем                                       |
| ------------ | ------------------------------- | ------------------------------------------- |
| `GET`        | `/api/me`                    | Есть ли юзер + профиль + прогресс           |
| `POST`       | `/api/users`                 | Создать пользователя                        |
| `POST`       | `/api/progress/reset`        | Сбросить прогресс выбранной роли            |
| `GET`        | `/api/training/current-step` | Текущий шаг обучения (`?role=`)             |
| `POST`       | `/api/training/answer`       | Ответ на шаг обучения                       |
| `GET`/`POST` | `/api/exam/start`            | Старт или восстановление экзамена (`?role=`) |
| `POST`       | `/api/exam/message`          | Сообщение в экзамене → ответ ИИ             |
| `POST`       | `/api/exam/finish`           | Завершить разговор и записать результат     |
| `POST`       | `/api/exam/restart`          | Начать экзамен заново                       |
| `GET`/`POST` | `/api/results`               | Результаты по роли                          |
| `GET`        | `/api/healthz`               | Проверка живости                            |
| `GET`        | `/api/docs`                  | Swagger UI                                  |
| `GET`        | `/api/openapi.yaml`          | Спецификация OpenAPI                        |
