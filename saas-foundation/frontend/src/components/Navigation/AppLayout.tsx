import { Outlet } from "react-router";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

/**
 * AppLayout is the shell for all authenticated pages.
 *
 * Structure:
 *   ┌─────────────────────────────────┐
 *   │  TopBar (h-14, full width)      │
 *   ├──────────┬──────────────────────┤
 *   │ Sidebar  │  <Outlet />          │
 *   │ (fixed w)│  (scrollable)        │
 *   └──────────┴──────────────────────┘
 *
 * The Sidebar manages its own collapsed/expanded state. The main content
 * area uses `flex-1` to fill the remaining horizontal space automatically.
 */
export function AppLayout() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />

      <div className="flex flex-col flex-1 min-w-0">
        <TopBar />

        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
