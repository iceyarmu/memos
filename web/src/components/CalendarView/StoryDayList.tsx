import { useQueries } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { storyServiceClient } from "@/connect";
import { storyKeys, useStories } from "@/hooks/useStories";
export const StoryDayList = ({ date }: { date: string }) => {
  const archives = useStories();
  const queries = useQueries({
    queries: (archives.data ?? []).map((s) => ({
      queryKey: [...storyKeys.all, s.name, "events"],
      queryFn: async () => (await storyServiceClient.listStoryEvents({ parent: s.name })).events,
    })),
  });
  const events = queries.flatMap((q) => q.data ?? []).filter((e) => e.eventDate === date);
  const error = archives.error ?? queries.find((q) => q.error)?.error;
  return (
    <section className="mt-4 border-t border-border pt-4">
      <h3 className="mb-2 font-medium">当天事迹</h3>
      {error ? (
        <p role="alert">事迹读取失败，请刷新重试。</p>
      ) : archives.isPending || queries.some((q) => q.isPending) ? (
        <p role="status">正在读取事迹…</p>
      ) : events.length === 0 ? (
        <p className="text-sm text-muted-foreground">当天没有事迹。</p>
      ) : (
        events.map((e) => (
          <Link
            key={e.name}
            to={`/stories?archive=${encodeURIComponent(e.story)}`}
            className="mb-2 block rounded-lg border border-border p-3"
          >
            <span className="text-xs text-muted-foreground">
              {archives.data?.find((s) => s.name === e.story)?.title} · {e.eventTime}
            </span>
            <p>{e.title || e.content.slice(0, 80)}</p>
          </Link>
        ))
      )}
    </section>
  );
};
