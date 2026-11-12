import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import StoryTimeline from "@/components/StoryTimeline";
import { storyServiceClient } from "@/connect";

const SharedStory = () => {
  const { token = "" } = useParams();
  const query = useQuery({
    queryKey: ["shared-story", token],
    queryFn: () => storyServiceClient.getSharedStory({ token }),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: true,
  });
  if (query.isPending)
    return (
      <p role="status" className="p-8">
        正在读取分享…
      </p>
    );
  if (query.error)
    return (
      <p role="alert" className="p-8">
        分享已撤销或无法读取。
      </p>
    );
  const { story, events } = query.data;
  const ordered = events.filter((e) => !e.parentEvent).flatMap((e) => [e, ...events.filter((a) => a.parentEvent === e.uid)]);
  return (
    <main className="mx-auto max-w-3xl space-y-5 p-5">
      <header>
        <p className="text-sm text-muted-foreground">只读人生档案</p>
        <h1 className="mt-2 text-2xl font-semibold">
          {story?.icon} {story?.title}
        </h1>
        <p className="mt-2 whitespace-pre-wrap">{story?.description}</p>
      </header>
      <StoryTimeline entries={ordered} color={story?.color} unit={story?.intervalUnit}>
        {(e) => (
          <article
            key={e.name}
            style={{ borderLeftColor: story?.color }}
            className="rounded-xl border border-border border-l-[3px] bg-card p-4 shadow-sm"
          >
            <time className="text-xs text-muted-foreground">
              {e.parentEvent ? "追加 · " : ""}
              {e.eventDate} {e.eventTime}
            </time>
            <h2 className="mt-2 break-words font-bold">{e.title}</h2>
            <p className="mt-2 whitespace-pre-wrap break-words">{e.content}</p>
            <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
              {e.attachmentUids.map((uid) => (
                <a
                  key={uid}
                  href={`/file/attachments/${encodeURIComponent(uid)}?story_share_token=${encodeURIComponent(token)}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  <img
                    alt="经历图片"
                    referrerPolicy="no-referrer"
                    className="aspect-square w-full rounded-lg object-cover"
                    src={`/file/attachments/${encodeURIComponent(uid)}?story_share_token=${encodeURIComponent(token)}`}
                  />
                </a>
              ))}
            </div>
          </article>
        )}
      </StoryTimeline>
    </main>
  );
};
export default SharedStory;
