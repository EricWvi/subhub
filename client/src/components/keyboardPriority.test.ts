import { describe, expect, it, vi } from "vitest";
import { claimKeyboardPriority } from "./keyboardPriority";

describe("claimKeyboardPriority", () => {
  it("keeps editor key events from reaching document shortcuts", () => {
    const editor = document.createElement("div");
    const input = document.createElement("textarea");
    const editorListener = vi.fn();
    const documentListener = vi.fn();
    editor.append(input);
    document.body.append(editor);
    for (const eventType of ["keydown", "keypress", "keyup"] as const) {
      input.addEventListener(eventType, editorListener);
      document.addEventListener(eventType, documentListener);
    }

    const release = claimKeyboardPriority(editor);

    for (const eventType of ["keydown", "keypress", "keyup"] as const) {
      for (const key of ["q", "e", "j"]) {
        input.dispatchEvent(
          new KeyboardEvent(eventType, {
            key,
            bubbles: true,
            cancelable: true,
          }),
        );
      }
    }

    expect(editorListener).toHaveBeenCalledTimes(9);
    expect(documentListener).not.toHaveBeenCalled();

    release();
    for (const eventType of ["keydown", "keypress", "keyup"] as const) {
      document.removeEventListener(eventType, documentListener);
    }
    editor.remove();
  });
});
