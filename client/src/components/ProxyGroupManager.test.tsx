import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../api";
import { renderWithProviders } from "../test/render";
import ProxyGroupManager from "./ProxyGroupManager";

vi.mock("@monaco-editor/react", async () => {
  const React = await import("react");
  const EditableEditor = ({
    onMount,
    value,
  }: {
    onMount?: (editor: {
      getDomNode: () => HTMLDivElement | null;
      getContainerDomNode: () => HTMLDivElement | null;
      onDidDispose: (callback: () => void) => void;
    }) => void;
    value?: string;
  }) => {
    const containerRef = React.useRef<HTMLDivElement>(null);
    const editorRef = React.useRef<HTMLDivElement>(null);

    React.useEffect(() => {
      const container = containerRef.current;
      if (!container) return;
      const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key !== "Backspace") return;
        const input = container.querySelector("textarea");
        input?.setRangeText(
          "",
          input.selectionStart,
          input.selectionEnd,
          "end",
        );
      };
      container.addEventListener("keydown", handleKeyDown);
      onMount?.({
        getDomNode: () => editorRef.current,
        getContainerDomNode: () => containerRef.current,
        onDidDispose: () => undefined,
      });
      return () => container.removeEventListener("keydown", handleKeyDown);
    }, []);

    return React.createElement(
      "div",
      { ref: containerRef },
      React.createElement(
        "div",
        { ref: editorRef },
        React.createElement("textarea", {
          "aria-label": "Script editor",
          readOnly: true,
          value,
        }),
      ),
    );
  };

  return {
    default: ({
      onMount,
      options,
      value,
    }: {
      onMount?: Parameters<typeof EditableEditor>[0]["onMount"];
      options?: { editContext?: boolean };
      value?: string;
    }) =>
      options?.editContext === false
        ? React.createElement(EditableEditor, { onMount, value })
        : React.createElement("div", {
            "aria-label": "Script editor",
            role: "textbox",
            tabIndex: 0,
          }),
  };
});

describe("ProxyGroupManager", () => {
  it("loads the configured script when editing a group directly", async () => {
    const user = userEvent.setup();
    const script = "function filter(nodes) { return nodes; }";
    const apiClient = vi.fn<ApiClient>(async () =>
      Response.json({
        groups: [
          {
            id: 1,
            name: "Streaming",
            script,
            created_at: "2026-09-02T05:05:09Z",
            updated_at: "2026-09-02T05:05:09Z",
          },
        ],
      }),
    );

    renderWithProviders(<ProxyGroupManager />, { apiClient });

    const row = (await screen.findByText("Streaming")).closest("tr");
    expect(row).not.toBeNull();
    await user.click(within(row!).getByRole("button", { name: "edit" }));

    const dialog = screen.getByRole("dialog", { name: "Edit Proxy Group" });
    const editor = within(dialog).getByRole<HTMLTextAreaElement>("textbox", {
      name: "Script editor",
    });
    expect(editor).toHaveValue(script);

    editor.setSelectionRange(0, script.length);
    editor.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: "Backspace",
        bubbles: true,
        cancelable: true,
      }),
    );
    expect(editor).toHaveValue("");
  });

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
