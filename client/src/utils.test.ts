import { describe, expect, it } from "vitest";
import { formatBytes, formatDate24h, getMonacoThemeMode } from "./utils";

describe("utility formatting", () => {
  it("formats timestamps with a 24-hour clock", () => {
    expect(formatDate24h("2026-09-02T05:05:09Z")).toBe(
      "2026-09-02 13:05:09",
    );
  });

  it("formats byte sizes", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("selects a Monaco theme from the background color", () => {
    expect(getMonacoThemeMode("#000")).toBe("vs-dark");
    expect(getMonacoThemeMode("#ffffff")).toBe("vs");
    expect(getMonacoThemeMode("invalid")).toBe("vs");
  });
});
