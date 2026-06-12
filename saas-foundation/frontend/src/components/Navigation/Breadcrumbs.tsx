import { useLocation, Link } from "react-router";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface BreadcrumbsProps {
  className?: string;
}

/**
 * Breadcrumbs — derives a path trail from the current URL.
 *
 * This is a stub implementation: it splits the pathname on "/" and capitalises
 * each segment. Task 2.7 will replace this with a route-aware implementation
 * that maps path segments to human-readable labels and supports dynamic IDs.
 */
export function Breadcrumbs({ className }: BreadcrumbsProps) {
  const { pathname } = useLocation();

  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) {
    return null;
  }

  return (
    <nav aria-label="Breadcrumb" className={cn("flex items-center", className)}>
      <ol className="flex items-center gap-1 text-sm text-muted-foreground">
        {segments.map((segment, index) => {
          const href = "/" + segments.slice(0, index + 1).join("/");
          const isLast = index === segments.length - 1;
          const label = segment.replace(/-/g, " ");

          return (
            <li key={href} className="flex items-center gap-1">
              {index > 0 && (
                <ChevronRight
                  className="h-3.5 w-3.5 shrink-0"
                  aria-hidden
                />
              )}
              {isLast ? (
                <span className="capitalize font-medium text-foreground">
                  {label}
                </span>
              ) : (
                <Link
                  to={href}
                  className="capitalize hover:text-foreground transition-colors"
                >
                  {label}
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
