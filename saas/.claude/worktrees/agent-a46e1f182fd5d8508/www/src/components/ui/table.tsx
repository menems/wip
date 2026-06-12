import { cn } from "../../lib/utils";

interface TableProps extends React.HTMLAttributes<HTMLTableElement> {}

export function Table({ className, ...props }: TableProps) {
  return (
    <div className="relative w-full overflow-auto">
      <table
        className={cn("w-full caption-bottom text-sm", className)}
        {...props}
      />
    </div>
  );
}

interface TableHeaderProps extends React.HTMLAttributes<HTMLTableSectionElement> {}

export function TableHeader({ className, ...props }: TableHeaderProps) {
  return <thead className={cn("[&_tr]:border-b", className)} {...props} />;
}

interface TableBodyProps extends React.HTMLAttributes<HTMLTableSectionElement> {}

export function TableBody({ className, ...props }: TableBodyProps) {
  return (
    <tbody className={cn("[&_tr:last-child]:border-0", className)} {...props} />
  );
}

interface TableFooterProps extends React.HTMLAttributes<HTMLTableSectionElement> {}

export function TableFooter({ className, ...props }: TableFooterProps) {
  return (
    <tfoot
      className={cn("border-t bg-muted/50 font-medium [&>tr]:last:border-b-0", className)}
      {...props}
    />
  );
}

interface TableRowProps extends React.HTMLAttributes<HTMLTableRowElement> {}

export function TableRow({ className, ...props }: TableRowProps) {
  return (
    <tr
      className={cn(
        "border-b border-border transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted",
        className
      )}
      {...props}
    />
  );
}

interface TableHeadProps extends React.ThHTMLAttributes<HTMLTableCellElement> {}

export function TableHead({ className, ...props }: TableHeadProps) {
  return (
    <th
      className={cn(
        "h-12 px-4 text-left align-middle font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0",
        className
      )}
      {...props}
    />
  );
}

interface TableCellProps extends React.TdHTMLAttributes<HTMLTableCellElement> {}

export function TableCell({ className, ...props }: TableCellProps) {
  return (
    <td
      className={cn("p-4 align-middle [&:has([role=checkbox])]:pr-0", className)}
      {...props}
    />
  );
}

interface TableCaptionProps extends React.HTMLAttributes<HTMLTableCaptionElement> {}

export function TableCaption({ className, ...props }: TableCaptionProps) {
  return (
    <caption
      className={cn("mt-4 text-sm text-muted-foreground", className)}
      {...props}
    />
  );
}
