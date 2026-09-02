import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../api";
import { renderWithProviders } from "../test/render";
import ProxyGroupManager from "./ProxyGroupManager";

vi.mock("@monaco-editor/react", async () => {
  const React = await import("react");
  return {
    default: ({ options }: { options?: { editContext?: boolean } }) =>
      options?.editContext === false
        ? React.createElement("textarea", { "aria-label": "Script editor" })
        : React.createElement("div", {
            "aria-label": "Script editor",
            role: "textbox",
            tabIndex: 0,
          }),
  };
});

describe("ProxyGroupManager", () => {
  it("uses an editable input target that keyboard extensions ignore", async () => {
    const user = userEvent.setup();
    const shortcutListener = vi.fn();
    const extensionListener = (event: KeyboardEvent) => {
      if (!(event.target instanceof HTMLTextAreaElement)) shortcutListener();
    };
    const apiClient = vi.fn<ApiClient>(async () =>
      Response.json({ groups: [] }),
    );
    window.addEventListener("keydown", extensionListener, true);

    try {
      renderWithProviders(<ProxyGroupManager />, { apiClient });
      await user.click(screen.getByRole("button", { name: /Add Group/ }));

      const editor = screen.getByRole("textbox", { name: "Script editor" });
      for (const key of ["q", "e", "j"]) {
        editor.dispatchEvent(
          new KeyboardEvent("keydown", {
            key,
            bubbles: true,
            cancelable: true,
          }),
        );
      }

      expect(shortcutListener).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener("keydown", extensionListener, true);
    }
  });
});
