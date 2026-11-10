export type PinKind = "project" | "task";

// A shortcut shown in the sidebar. The label is a snapshot taken when pinning
// and refreshed on rename, so the sidebar can render it without a request.
export type Pin = {
  kind: PinKind;
  id: string;
  label: string;
  href: string;
  // For task pins: the owning project, so deleting it drops the pin too.
  projectId?: string;
};
