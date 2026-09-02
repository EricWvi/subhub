const keyboardEventTypes = ["keydown", "keypress", "keyup"] as const;

export const claimKeyboardPriority = (root: HTMLElement) => {
  const stopPropagation = (event: KeyboardEvent) => event.stopPropagation();

  for (const eventType of keyboardEventTypes) {
    root.addEventListener(eventType, stopPropagation);
  }

  return () => {
    for (const eventType of keyboardEventTypes) {
      root.removeEventListener(eventType, stopPropagation);
    }
  };
};
