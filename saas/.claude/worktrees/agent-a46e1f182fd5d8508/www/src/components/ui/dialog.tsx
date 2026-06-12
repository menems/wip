import { useEffect, useRef, useCallback } from "react";
import { cn } from "../../lib/utils";

interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

export function Dialog({ open, onOpenChange, children }: DialogProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onOpenChange(false);
      }
    }
    if (open) {
      document.addEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [open, onOpenChange]);

  const handleOverlayClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === overlayRef.current) {
        onOpenChange(false);
      }
    },
    [onOpenChange]
  );

  if (!open) return null;

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
      onClick={handleOverlayClick}
      role="presentation"
    >
      {children}
    </div>
  );
}

interface DialogContentProps extends React.HTMLAttributes<HTMLDivElement> {}

export function DialogContent({ className, children, ...props }: DialogContentProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className={cn(
        "relative z-50 grid w-full max-w-lg gap-4 rounded-lg border border-border bg-background p-6 shadow-lg",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

interface DialogHeaderProps extends React.HTMLAttributes<HTMLDivElement> {}

export function DialogHeader({ className, ...props }: DialogHeaderProps) {
  return (
    <div
      className={cn("flex flex-col space-y-1.5 text-center sm:text-left", className)}
      {...props}
    />
  );
}

interface DialogFooterProps extends React.HTMLAttributes<HTMLDivElement> {}

export function DialogFooter({ className, ...props }: DialogFooterProps) {
  return (
    <div
      className={cn("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2", className)}
      {...props}
    />
  );
}

interface DialogTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {}

export function DialogTitle({ className, ...props }: DialogTitleProps) {
  return (
    <h2
      className={cn("text-lg font-semibold leading-none tracking-tight", className)}
      {...props}
    />
  );
}

interface DialogDescriptionProps extends React.HTMLAttributes<HTMLParagraphElement> {}

export function DialogDescription({ className, ...props }: DialogDescriptionProps) {
  return (
    <p className={cn("text-sm text-muted-foreground", className)} {...props} />
  );
}
