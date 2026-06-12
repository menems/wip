import { useState } from "react";
import { NavLink } from "react-router";
import {
  LayoutDashboard,
  Users,
  Shield,
  FileText,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/features/auth/useAuth";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

// ---------------------------------------------------------------------------
// Nav item definitions
// ---------------------------------------------------------------------------

interface NavItem {
  to: string;
  label: string;
  /** Lucide icon component */
  icon: React.ComponentType<{ className?: string; "aria-hidden"?: boolean }>;
  /** "resource:action" permission required, or null for any authenticated user. */
  permission: string | null;
}

const NAV_ITEMS: NavItem[] = [
  {
    to: "/dashboard",
    label: "Dashboard",
    icon: LayoutDashboard,
    permission: null,
  },
  { to: "/users", label: "Users", icon: Users, permission: "users:read" },
  { to: "/roles", label: "Roles", icon: Shield, permission: "roles:read" },
  {
    to: "/audit-logs",
    label: "Audit Logs",
    icon: FileText,
    permission: "audit_logs:read",
  },
];

// ---------------------------------------------------------------------------
// NavItem sub-component
// ---------------------------------------------------------------------------

interface NavItemProps {
  item: NavItem;
  isCollapsed: boolean;
}

function SidebarNavItem({ item, isCollapsed }: NavItemProps) {
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      "flex items-center gap-3 rounded-md py-2 text-sm transition-colors",
      "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
      isActive &&
        "bg-sidebar-accent text-sidebar-accent-foreground font-medium",
      isCollapsed ? "justify-center px-2" : "px-3"
    );

  const link = (
    <NavLink to={item.to} className={linkClass} end={item.to === "/dashboard"}>
      <item.icon className="h-4 w-4 shrink-0" aria-hidden />
      {!isCollapsed && <span>{item.label}</span>}
    </NavLink>
  );

  if (!isCollapsed) return link;

  // When collapsed, wrap in a tooltip so users can still identify the link.
  return (
    <Tooltip>
      <TooltipTrigger asChild>{link}</TooltipTrigger>
      <TooltipContent side="right">{item.label}</TooltipContent>
    </Tooltip>
  );
}

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

/**
 * Sidebar renders the primary navigation links.
 *
 * - Collapses to icon-only mode on toggle (state is local; persisted to
 *   localStorage in task 2.7).
 * - Links that require a permission the current user lacks are hidden.
 * - Active link is highlighted via React Router's NavLink `isActive` prop.
 */
export function Sidebar() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const { hasPermission } = useAuth();

  const visibleItems = NAV_ITEMS.filter((item) => {
    if (item.permission === null) return true;
    const [resource, action] = item.permission.split(":");
    return hasPermission(resource ?? "", action ?? "");
  });

  return (
    <TooltipProvider delayDuration={200}>
      <aside
        aria-label="Main navigation"
        className={cn(
          "flex flex-col border-r bg-sidebar transition-all duration-200 ease-in-out",
          isCollapsed ? "w-16" : "w-64"
        )}
      >
        {/* Brand / logo area */}
        <div
          className={cn(
            "flex h-14 items-center border-b shrink-0",
            isCollapsed ? "justify-center px-2" : "px-4"
          )}
        >
          {isCollapsed ? (
            <Shield className="h-5 w-5 text-sidebar-primary" aria-hidden />
          ) : (
            <span className="font-semibold text-sm tracking-tight text-sidebar-foreground">
              SaaS Foundation
            </span>
          )}
        </div>

        {/* Navigation links */}
        <nav className="flex-1 space-y-0.5 p-2">
          {visibleItems.map((item) => (
            <SidebarNavItem
              key={item.to}
              item={item}
              isCollapsed={isCollapsed}
            />
          ))}
        </nav>

        {/* Collapse toggle */}
        <div className="border-t p-2">
          <button
            onClick={() => setIsCollapsed((prev) => !prev)}
            aria-label={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
            className={cn(
              "flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm",
              "text-sidebar-foreground hover:bg-sidebar-accent transition-colors",
              isCollapsed && "justify-center px-2"
            )}
          >
            {isCollapsed ? (
              <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4 shrink-0" aria-hidden />
                <span>Collapse</span>
              </>
            )}
          </button>
        </div>
      </aside>
    </TooltipProvider>
  );
}
