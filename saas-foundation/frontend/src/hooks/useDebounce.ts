import { useEffect, useState } from "react";

/**
 * useDebounce delays propagating a value change until the user has stopped
 * updating it for `delay` milliseconds. Used to avoid firing a query on
 * every keypress in search inputs.
 */
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debouncedValue;
}
