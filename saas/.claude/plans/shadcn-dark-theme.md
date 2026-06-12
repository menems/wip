# Plan: shadcn-dark-theme
> Install shadcn/ui and apply a dark theme across the app using CSS variables and the `dark` class strategy

**Created**: 2026-03-11  |  **Branch**: feat/shadcn-dark-theme  |  **Status**: in-progress

## Steps

- [x] `step-01` [frontend] Install shadcn dependencies and scaffold configuration
  - **Files**: www/package.json (modify), www/components.json (create), www/src/lib/utils.ts (create)
  - **Commit**: `chore(frontend): install shadcn deps and scaffold components.json + cn util`
  - **Approval**: required
  - Installs `tailwind-merge` and `class-variance-authority` (shadcn's required peer deps). Creates `components.json` configured for Tailwind v4 with dark mode set to `class`. Creates the `cn()` utility at `www/src/lib/utils.ts` combining `clsx` + `tailwind-merge`.
  - **Done when**: `npm install` succeeds; `cn()` is importable and correctly merges/deduplicates Tailwind classes

- [x] `step-02` [frontend] Add CSS variable tokens for light and dark theme
  - **Files**: www/src/styles.css (modify)
  - **Commit**: `feat(frontend): add shadcn CSS variable tokens for light and dark theme`
  - Adds the shadcn CSS variable palette to `styles.css`: `:root` block for light theme and `.dark` block for dark theme. Covers `--background`, `--foreground`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--ring`, and radius tokens. Adds `@layer base` with the shadcn base reset (`border-border`, `bg-background`, `text-foreground`).
  - **Done when**: CSS variables resolve correctly in browser; switching `.dark` class on `<html>` visually changes the palette

- [x] `step-03` [frontend] Add shadcn Button, Input, Label, and Card components
  - **Files**: www/src/components/ui/button.tsx (create), www/src/components/ui/input.tsx (create), www/src/components/ui/label.tsx (create), www/src/components/ui/card.tsx (create)
  - **Commit**: `feat(frontend): add shadcn Button, Input, Label, Card UI components`
  - Scaffolds the four shadcn components needed to rebuild the login page. `Button` uses `cva` for size/variant variants. `Input` wraps a native `<input>` with consistent ring/border styling. `Label` wraps native `<label>`. `Card` provides `Card`, `CardHeader`, `CardContent`, and `CardTitle` sub-components. All use CSS variable-based colors and `cn()` for className merging.
  - **Done when**: All four components render without errors; TypeScript compiles cleanly with no `any` usage

- [x] `step-04` [frontend] Refactor LoginPage to use shadcn components
  - **Files**: www/src/routes/login.tsx (modify), www/src/routes/login.test.tsx (modify)
  - **Commit**: `refactor(frontend): migrate LoginPage to shadcn components with dark theme`
  - Replaces manual Tailwind utility strings in `login.tsx` with the new `<Button>`, `<Input>`, `<Label>`, and `<Card>` shadcn components. Removes hand-rolled `dark:` class variants now covered by CSS variables. Updates tests to pass with the new component structure (semantic roles remain, so most assertions stay unchanged).
  - **Done when**: Login page renders with dark theme applied; all existing tests pass; no raw `dark:bg-gray-*` classes remain in the file

- [x] `step-05` [frontend] Enable dark mode globally via root element
  - **Files**: www/src/routes/__root.tsx (modify)
  - **Commit**: `feat(frontend): enable dark mode by default via dark class on html root`
  - Adds the `dark` class to the document root (`document.documentElement`) or a wrapping element in the root route so the dark theme is active app-wide. Implements as a default-dark approach (class applied on mount) with a `useEffect` in the root component, or simply via an inline `className="dark"` on a full-height wrapper.
  - **Done when**: Entire app renders in dark theme on load; all `dark:` variants and CSS variable `.dark` overrides activate correctly

## Log

- [2026-03-11 19:00] step-01 done by frontend — chore(frontend): install shadcn deps and scaffold components.json + cn util
- [2026-03-11] step-02 done by frontend — feat(frontend): add shadcn CSS variable tokens for light and dark theme
- [2026-03-11 19:07] step-03 done by frontend — feat(frontend): add shadcn Button, Input, Label, Card UI components
- [2026-03-11 19:10] step-04 done by frontend — refactor(frontend): migrate LoginPage to shadcn components with dark theme
- [2026-03-11 19:12] step-05 done by frontend — feat(frontend): enable dark mode by default via dark class on html root

## Notes
- Tailwind CSS v4 is CSS-first — no `tailwind.config.js` needed; all theme tokens go in `styles.css`
- shadcn `components.json` must set `"tailwind": { "version": "4" }` (or equivalent) for v4 compatibility
- `clsx` is already installed — `cn()` just adds `tailwind-merge` on top
- Dark mode strategy: `class` (not `media`) — gives explicit control over when dark applies
