import { useNavigate } from "react-router";
import { LogOut } from "lucide-react";
import { useAuth } from "@/features/auth/useAuth";
import { UserAvatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Breadcrumbs } from "./Breadcrumbs";

/**
 * TopBar renders the application header.
 *
 * Left side: Breadcrumbs navigation trail.
 * Right side: User avatar + dropdown (name, email, sign-out).
 *
 * A theme toggle will be added in task 2.8.
 */
export function TopBar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  async function handleLogout() {
    await logout();
    void navigate("/login", { replace: true });
  }

  return (
    <header className="flex h-14 shrink-0 items-center border-b bg-background px-6 gap-4">
      {/* Breadcrumb trail fills the available space */}
      <Breadcrumbs className="flex-1 min-w-0" />

      {/* User menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="flex items-center gap-2 px-2"
            aria-label="User menu"
          >
            <UserAvatar name={user?.name ?? "?"} />
            <span className="hidden sm:block text-sm font-normal max-w-[160px] truncate">
              {user?.name}
            </span>
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="end" className="w-52">
          <DropdownMenuLabel className="flex flex-col gap-0.5">
            <span className="font-medium">{user?.name}</span>
            <span className="text-xs font-normal text-muted-foreground truncate">
              {user?.email}
            </span>
          </DropdownMenuLabel>

          <DropdownMenuSeparator />

          <DropdownMenuItem
            onClick={() => void handleLogout()}
            className="cursor-pointer"
          >
            <LogOut className="mr-2 h-4 w-4" aria-hidden />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
