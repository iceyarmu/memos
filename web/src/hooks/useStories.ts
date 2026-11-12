import { useQueries, useQuery } from "@tanstack/react-query";
import { storyServiceClient } from "@/connect";
export const storyKeys = { all: ["stories"] as const };
export const useStories = (enabled = true) =>
  useQuery({ queryKey: storyKeys.all, enabled, queryFn: async () => (await storyServiceClient.listStories({})).stories });
export const useStoryEvents = (parent?: string) =>
  useQuery({
    queryKey: [...storyKeys.all, parent, "events"],
    enabled: Boolean(parent),
    queryFn: async () => (await storyServiceClient.listStoryEvents({ parent })).events,
  });

export const useStoryDateCounts = (enabled: boolean) => {
  const archives = useStories(enabled);
  const queries = useQueries({
    queries: enabled
      ? (archives.data ?? []).map((s) => ({
          queryKey: [...storyKeys.all, s.name, "events"],
          queryFn: async () => (await storyServiceClient.listStoryEvents({ parent: s.name })).events,
        }))
      : [],
  });
  const counts: Record<string, number> = {};
  if (enabled) for (const q of queries) for (const e of q.data ?? []) counts[e.eventDate] = (counts[e.eventDate] ?? 0) + 1;
  return counts;
};
