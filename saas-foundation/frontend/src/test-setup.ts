import "@testing-library/jest-dom";

// ---------------------------------------------------------------------------
// ResizeObserver polyfill
// Radix UI components (Checkbox, Select, etc.) use @radix-ui/react-use-size
// which calls ResizeObserver. jsdom does not implement it, so we provide a
// no-op stub so the components mount without errors in tests.
// ---------------------------------------------------------------------------
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
