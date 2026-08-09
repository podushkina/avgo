# AGENTS.md

Instructions for AI agents working on this frontend.

## Stack

- React 19 + TypeScript
- Vite
- Vanilla Extract (`*.css.ts`, zero-runtime CSS-in-JS)
- React Router (`react-router`)
- MobX (`RootStore` singleton + feature stores, `mobx-react-lite`)
- ESLint + Prettier
- Yarn (`packageManager` in `package.json`)

## `src` structure

```
src/
  App/           # root UI, router, app-level wiring
  configs/       # app-level config (ROUTES, …)
  mocks/         # MSW browser worker, handlers, in-memory db, fixture data
  stores/        # MobX stores (RootStore singleton, UserStore, …)
  pages/         # route pages; connect to the store here
  layouts/       # shells: BaseLayout, Header, Footer
  shared/        # reusable domain-free UI + api client
  index.tsx      # createRoot entry (starts MSW in DEV)
```

- Do not create both `src/app` and `src/App` — macOS is case-insensitive and they collide.
- Pages compose layouts and shared UI; shared components stay presentational (props in, no store access).

## Components

Every React component lives in its own folder with a public `index.ts` barrel:

```
ComponentName/
  ComponentName.tsx
  ComponentName.css.ts   # only if styles are needed (Vanilla Extract)
  index.ts               # required
```

`index.ts` example:

```ts
export { default } from './ComponentName';
```

Rules:

- Always create the folder + `index.ts` when adding a component.
- Import via the folder / barrel (`from './Header'`, `from '../layouts'`), not deep into `ComponentName.tsx` from outside.
- Colocate Vanilla Extract styles as `ComponentName.css.ts` next to the component.

## State

- Global MobX `RootStore` singleton under `stores/RootStore/` (`rootStore`).
- Domain stores are fields on `RootStore` (e.g. `user: UserStore` in `stores/UserStore/`).
- Import the `rootStore` singleton directly; wrap reactive UI with `observer`.
- Connect to the store in `pages/` (and layout pieces that truly need it).
- Keep `shared/` components free of store access; pass data via props.

## Mocks / API

- Stores talk to the backend via `fetch` (`src/shared/api`) on `/api/*` paths. Do not put mock responses or artificial `setTimeout` delays inside stores.
- In local dev, MSW (`src/mocks/`) intercepts `/api/*`: handlers + in-memory `db` + fixture data under `src/mocks/data/`. Delays (~300–800ms) live in MSW handlers via `delay()` from `msw`.
- Drive UI from `LoadingStageModel` (`loading` → `success` / `error`) around every API call.
- MSW starts from `src/index.tsx` when `import.meta.env.DEV` and `VITE_ENABLE_MSW` is not `'false'`. Set `VITE_ENABLE_MSW=false` to hit a real backend through the Vite `/api` proxy.
- Keep fixture payloads in `src/mocks/data/`; do not inline large mock payloads in stores or handlers.

## Commands

- `yarn dev` — local dev server
- `yarn build` — production build
- `yarn lint` / `yarn lint:fix` — ESLint
