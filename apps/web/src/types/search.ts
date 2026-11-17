import type { ResourceKind } from "./resource";

export type SearchProject = {
  id: string;
  name: string;
  icon: string;
};

export type SearchPlan = {
  id: string;
  projectId: string;
  projectName: string;
  name: string;
  icon: string;
};

export type SearchTask = {
  id: string;
  projectId: string;
  projectName: string;
  planId: string;
  planName: string;
  // Set for subtasks; they open through their parent task.
  parentId: string | null;
  title: string;
  done: boolean;
};

export type SearchResource = {
  id: string;
  parentId: string | null;
  // The folder the item sits in; null at the top of the tree.
  parentName: string | null;
  kind: ResourceKind;
  name: string;
  mimeType: string;
  isText: boolean;
};

export type SearchResults = {
  projects: SearchProject[];
  plans: SearchPlan[];
  tasks: SearchTask[];
  resources: SearchResource[];
};
