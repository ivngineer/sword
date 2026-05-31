import { create } from "zustand";

// Shape of one in-flight install/remove. fraction is null when the underlying
// tool has not yet emitted a parseable percentage (we render an indeterminate
// bar in that case); status is the most recent line from the tool, used as a
// short label under the bar.
export type ProgressJob = {
  id: string;
  kind: "install" | "remove";
  appName: string;
  sourceType: string;
  fraction: number | null;
  status: string;
};

type State = {
  jobs: Record<string, ProgressJob>;
  start: (job: Omit<ProgressJob, "fraction" | "status">) => void;
  update: (id: string, fraction: number | null, status: string) => void;
  finish: (id: string) => void;
};

export const useProgressStore = create<State>((set) => ({
  jobs: {},
  start: (job) =>
    set((s) => ({
      jobs: { ...s.jobs, [job.id]: { ...job, fraction: null, status: "" } },
    })),
  update: (id, fraction, status) =>
    set((s) => {
      const cur = s.jobs[id];
      if (!cur) return s;
      return { jobs: { ...s.jobs, [id]: { ...cur, fraction, status } } };
    }),
  finish: (id) =>
    set((s) => {
      if (!s.jobs[id]) return s;
      const next = { ...s.jobs };
      delete next[id];
      return { jobs: next };
    }),
}));

// activeJob returns the most recent job (or null if none). The sidebar
// shows one bar at a time; when multiple actions are running we pick the
// freshest by insertion order in the map.
export function activeJob(jobs: Record<string, ProgressJob>): ProgressJob | null {
  const keys = Object.keys(jobs);
  if (keys.length === 0) return null;
  return jobs[keys[keys.length - 1]];
}
