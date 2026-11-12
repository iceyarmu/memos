import { LinkIcon } from "lucide-react";
import { Link } from "react-router-dom";
import { useMemo } from "@/hooks/useMemoQueries";

export default function StoryMemoLink({ name, from }: { name: string; from: string }) {
  const query = useMemo(name);
  return (
    <div className="mt-3 rounded-lg border border-border/70 bg-muted/30 px-3 py-2 text-sm">
      {query.isLoading ? (
        <p className="text-muted-foreground">正在读取关联 Memo…</p>
      ) : query.data ? (
        <Link to={`/${name}`} state={{ from }} className="block rounded-sm focus-visible:outline-2 focus-visible:outline-ring">
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <LinkIcon className="size-3.5" />
            关联 Memo · 点击查看
          </span>
          <p className="mt-1 line-clamp-2 break-words">{query.data.snippet || "图片或附件 Memo"}</p>
        </Link>
      ) : (
        <div className="text-muted-foreground">
          <p>{query.isUnavailable ? "关联 Memo 已删除或无法访问" : "关联 Memo 暂时无法读取"}</p>
          {!query.isUnavailable && (
            <button type="button" className="mt-1 underline" onClick={() => void query.refetch()}>
              重试
            </button>
          )}
        </div>
      )}
    </div>
  );
}
