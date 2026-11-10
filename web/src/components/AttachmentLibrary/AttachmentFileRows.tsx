import { DownloadIcon, FileAudioIcon, FileIcon, PlayIcon } from "lucide-react";
import AudioAttachmentItem from "@/components/MemoMetadata/Attachment/AudioAttachmentItem";
import VideoPoster from "@/components/VideoPoster";
import type { AttachmentLibraryListItem } from "@/hooks/useAttachmentLibrary";
import { cn } from "@/lib/utils";
import type { User } from "@/types/proto/api/user_service_pb";
import {
  getAttachmentOpenUrl,
  getAttachmentThumbnailUrl,
  getAttachmentType,
  getHTMLAttachmentDownloadUrl,
  isHTMLAttachment,
  isMotionAttachment,
} from "@/utils/attachment";
import { AttachmentCreator, AttachmentMetadataLine, AttachmentOpenButton, AttachmentSourceChip } from "./AttachmentLibraryPrimitives";

type AttachmentCreators = Map<string, User | undefined>;

const FileActions = ({ item }: { item: AttachmentLibraryListItem }) => (
  <>
    {isHTMLAttachment(item.attachment) && (
      <a
        href={getHTMLAttachmentDownloadUrl(item.attachment)}
        download
        aria-label={`Download ${item.attachment.filename}`}
        className="p-1.5 text-muted-foreground hover:text-foreground"
      >
        <DownloadIcon className="size-3.5" />
      </a>
    )}
    <AttachmentOpenButton href={getAttachmentOpenUrl(item.attachment)} />
  </>
);

const AttachmentThumb = ({ item, className }: { item: AttachmentLibraryListItem; className?: string }) => {
  const type = getAttachmentType(item.attachment);
  const isMotion = isMotionAttachment(item.attachment);

  if (type === "image/*" || isMotion) {
    return (
      <div className={cn("overflow-hidden rounded-xl bg-muted/35", className)}>
        <img
          src={getAttachmentThumbnailUrl(item.attachment)}
          alt={item.attachment.filename}
          className="h-full w-full object-cover"
          loading="lazy"
          decoding="async"
        />
      </div>
    );
  }

  if (type === "video/*") {
    return (
      <div className={cn("relative overflow-hidden rounded-xl bg-muted/35", className)}>
        <VideoPoster sourceUrl={item.sourceUrl} alt={item.attachment.filename} className="h-full w-full object-cover" />
        <span className="absolute bottom-2 right-2 inline-flex h-7 w-7 items-center justify-center rounded-full bg-background/85 text-foreground shadow-sm">
          <PlayIcon className="h-3.5 w-3.5 fill-current" />
        </span>
      </div>
    );
  }

  return (
    <div className={cn("flex items-center justify-center rounded-xl bg-muted/45 text-muted-foreground", className)}>
      {type === "audio/*" ? <FileAudioIcon className="h-5 w-5" /> : <FileIcon className="h-5 w-5" />}
    </div>
  );
};

export const AttachmentDocumentRows = ({ items, creators }: { items: AttachmentLibraryListItem[]; creators?: AttachmentCreators }) => {
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <article
          key={item.attachment.name}
          className="flex items-center gap-2.5 rounded-[18px] border border-border/60 bg-background/90 p-3 shadow-sm shadow-black/[0.02]"
        >
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted/45 text-muted-foreground">
            <FileIcon className="h-4.5 w-4.5" />
          </div>

          <div className="min-w-0 flex-1">
            <a
              href={getAttachmentOpenUrl(item.attachment)}
              target="_blank"
              rel="noreferrer"
              className="block truncate text-sm font-medium text-foreground hover:underline"
              title={item.attachment.filename}
            >
              {item.attachment.filename}
            </a>
            <div className="mt-0.5 flex flex-wrap items-center gap-1.5">
              <AttachmentMetadataLine className="min-w-0 max-w-full" items={[item.fileTypeLabel, item.fileSizeLabel, item.createdLabel]} />
              <AttachmentCreator creatorName={item.attachment.creator} user={creators?.get(item.attachment.creator)} />
              <AttachmentSourceChip memoName={item.memoName} />
            </div>
          </div>

          <FileActions item={item} />
        </article>
      ))}
    </div>
  );
};

export const AttachmentAudioRows = ({ items, creators }: { items: AttachmentLibraryListItem[]; creators?: AttachmentCreators }) => {
  return (
    <div className="space-y-2.5">
      {items.map((item) => (
        <article
          key={item.attachment.name}
          className="rounded-[18px] border border-border/60 bg-background/90 p-2.5 shadow-sm shadow-black/[0.02]"
        >
          <AudioAttachmentItem
            filename={item.attachment.filename}
            sourceUrl={item.sourceUrl}
            mimeType={item.attachment.type}
            size={Number(item.attachment.size)}
          />
          <div className="mt-2.5 flex items-center justify-between gap-2 border-t border-border/60 px-0.5 pt-2.5">
            <div className="min-w-0 flex flex-wrap items-center gap-1.5">
              <AttachmentMetadataLine className="min-w-0 max-w-full" items={[item.createdLabel]} />
              <AttachmentCreator creatorName={item.attachment.creator} user={creators?.get(item.attachment.creator)} />
              <AttachmentSourceChip memoName={item.memoName} />
            </div>
            <AttachmentOpenButton href={item.sourceUrl} />
          </div>
        </article>
      ))}
    </div>
  );
};

export const AttachmentUnusedRows = ({ items, creators }: { items: AttachmentLibraryListItem[]; creators?: AttachmentCreators }) => {
  return (
    <div className="space-y-2.5">
      {items.map((item) => (
        <article
          key={item.attachment.name}
          className="flex items-center gap-2.5 rounded-[18px] border border-warning/30 bg-warning/5 p-3 shadow-sm shadow-black/[0.02]"
        >
          <AttachmentThumb item={item} className="h-10 w-10 shrink-0" />

          <div className="min-w-0 flex-1">
            <a
              href={getAttachmentOpenUrl(item.attachment)}
              target="_blank"
              rel="noreferrer"
              className="block truncate text-sm font-medium text-foreground hover:underline"
              title={item.attachment.filename}
            >
              {item.attachment.filename}
            </a>
            <div className="mt-0.5 flex flex-wrap items-center gap-1.5">
              <AttachmentMetadataLine className="min-w-0 max-w-full" items={[item.fileTypeLabel, item.fileSizeLabel, item.createdLabel]} />
              <AttachmentCreator creatorName={item.attachment.creator} user={creators?.get(item.attachment.creator)} />
              <AttachmentSourceChip unlinkedLabelKey="attachment-library.labels.not-linked" />
            </div>
          </div>

          <FileActions item={item} />
        </article>
      ))}
    </div>
  );
};
