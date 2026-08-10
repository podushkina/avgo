# Frontend — Безопасная сделка

## Стек

- React 19, TypeScript
- Vite
- React Router
- MobX
- Tailwind CSS, shadcn/ui
- MSW (моки API в dev)
- ESLint, Prettier
- Yarn 4

## Команды

```bash
yarn install          # зависимости
yarn dev              # dev-сервер на http://localhost:3000
yarn build            # production-сборка в build/
yarn preview          # локальный просмотр сборки
yarn lint             # ESLint
yarn lint:fix       # ESLint с автофиксом
```

В `yarn dev` по умолчанию поднимается MSW. Чтобы ходить в реальный бэкенд через Vite-прокси `/api` → `localhost:8081`, отключи моки так:

```bash
VITE_ENABLE_MSW=false yarn dev
```

или добавь в `.env.local`:

```bash
VITE_ENABLE_MSW=false
```
