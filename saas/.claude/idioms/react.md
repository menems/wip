# React/TypeScript Conventions

## Stack
React 19, TanStack Router + Query, Vite, Tailwind CSS 4, TypeScript, ConnectRPC

## API Clients
- API clients are **generated from proto files** — never hand-write HTTP/fetch calls
- Use `@connectrpc/connect-web` + `@connectrpc/connect-query` for TanStack Query integration
- Import generated service definitions from the generated code directory
- Proto changes are a backend concern — frontend consumes the generated output
- Run code generation after proto files change (e.g., `buf generate`)

## Component Patterns
- One component per file, named export matching filename
- All props defined as TypeScript `interface` (not `type`), co-located above component
- Composition over prop drilling — use context or TanStack Query for shared state
- Prefer `function` declarations over arrow functions for components
- Co-locate tests (`Component.test.tsx`) next to component

## TanStack Router
- File-based routing in `www/src/routes/`
- Loaders for data fetching — no `useEffect` for initial data
- Search params validated with schema (zod/valibot)
- Lazy routes for code splitting

## TanStack Query
- All server state through `useQuery` / `useMutation` — no local state for API data
- Use `@connectrpc/connect-query` hooks (`useQuery` with generated descriptors) for type-safe API calls
- Custom hooks per query: `useUser()`, `useUpdateUser()` — wrap connect-query hooks, never raw in components
- Query keys derived from generated service descriptors — no manual key management
- Optimistic updates for mutations where UX demands it
- `queryClient.invalidateQueries` after mutations, not manual cache updates

## Tailwind CSS 4
- Utility-first, no custom CSS unless unavoidable
- Extract repeated patterns into components, not `@apply`
- Responsive: mobile-first (`sm:`, `md:`, `lg:`)
- Dark mode via `dark:` variant
- Use `cn()` or `clsx` for conditional classes

## Accessibility
- Semantic HTML elements (`button`, `nav`, `main`, `section`)
- ARIA attributes only when semantic HTML is insufficient
- Keyboard navigation: all interactive elements focusable + operable
- Focus management on route transitions

## Testing
- Vitest + Testing Library
- Test user behavior, not implementation details
- `screen.getByRole` preferred over `getByTestId`
- Mock API calls at the network level (MSW)
- Co-located test files

## Idioms
- No `any` — use `unknown` + type narrowing
- Prefer `satisfies` over `as` for type assertions
- Custom hooks for reusable logic (`use` prefix)
- No barrel files (`index.ts` re-exports) — direct imports
- Minimal diffs — do not rewrite unchanged code
