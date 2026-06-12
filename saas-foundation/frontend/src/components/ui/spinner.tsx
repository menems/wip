import { cn } from "@/lib/utils";

interface SpinnerProps {
  className?: string;
  /** Accessible label for screen readers. Defaults to "Loading…" */
  label?: string;
}

/**
 * Spinner — animated loading indicator.
 * Renders a spinning ring using Tailwind's animate-spin utility.
 */
export function Spinner({ className, label = "Loading…" }: SpinnerProps) {
  return (
    <div
      role="status"
      aria-label={label}
      className={cn(
        "inline-block h-6 w-6 animate-spin rounded-full border-2 border-current border-t-transparent text-primary",
        className
      )}
    />
  );
}
