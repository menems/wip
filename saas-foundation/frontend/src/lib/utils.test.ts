import { describe, it, expect } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("deduplicates conflicting Tailwind classes", () => {
    // tailwind-merge keeps the last conflicting class
    expect(cn("p-4", "p-6")).toBe("p-6");
  });

  it("handles conditional classes", () => {
    expect(cn("base", false && "skip", "end")).toBe("base end");
  });

  it("handles undefined values", () => {
    expect(cn("base", undefined, "end")).toBe("base end");
  });
});
