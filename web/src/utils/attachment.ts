import { Attachment, MotionMediaFamily, MotionMediaRole } from "@/types/proto/api/attachment_service_pb";

const isIPAddress = (hostname: string): boolean => /^(\d{1,3}\.){3}\d{1,3}$/.test(hostname) || hostname.includes(":");

export const isHTMLAttachment = (attachment: Attachment): boolean => {
  if (attachment.type.split(";")[0].trim().toLowerCase() !== "text/html") return false;
  if (!attachment.externalLink) return true;
  try {
    const url = new URL(attachment.externalLink, window.location.origin);
    return url.origin === window.location.origin && url.pathname.startsWith("/file/attachments/") && url.searchParams.has("share_token");
  } catch {
    return false;
  }
};

// Preview and download use the authorized endpoint, including on hosts with a /files/ mapping.
export const getHTMLAttachmentDownloadUrl = (attachment: Attachment): string => {
  const url = new URL(`/file/${attachment.name}/${encodeURIComponent(attachment.filename)}`, window.location.origin);
  if (attachment.externalLink) {
    const external = new URL(attachment.externalLink, window.location.origin);
    const token = external.searchParams.get("share_token");
    if (external.origin === url.origin && token) url.searchParams.set("share_token", token);
  }
  return url.toString();
};

export const getAttachmentOpenUrl = (attachment: Attachment): string => {
  if (!isHTMLAttachment(attachment)) return getAttachmentUrl(attachment);
  const url = new URL(getHTMLAttachmentDownloadUrl(attachment));
  url.searchParams.set("preview", "true");
  return url.toString();
};

export const getAttachmentUrl = (attachment: Attachment) => {
  if (attachment.externalLink) {
    return attachment.externalLink;
  }

  if (attachment.reference && !isIPAddress(window.location.hostname)) {
    return `${window.location.origin}/files/${attachment.reference}`;
  }

  return `${window.location.origin}/file/${attachment.name}/${encodeURIComponent(attachment.filename)}`;
};

// Appends a thumbnail or motion selector only to share-mode links; other external links fall back to the server-managed attachment URL.
const withShareTokenParam = (externalLink: string | undefined, key: string): string | undefined => {
  if (!externalLink) {
    return undefined;
  }
  const url = new URL(externalLink, window.location.origin);
  if (!url.searchParams.has("share_token")) {
    return undefined;
  }
  url.searchParams.set(key, "true");
  return url.toString();
};

export const getAttachmentThumbnailUrl = (attachment: Attachment) => {
  const shareUrl = withShareTokenParam(attachment.externalLink, "thumbnail");
  if (shareUrl) {
    return shareUrl;
  }

  return `${window.location.origin}/file/${attachment.name}/${encodeURIComponent(attachment.filename)}?thumbnail=true`;
};

export const getAttachmentMotionClipUrl = (attachment: Attachment) => {
  const shareUrl = withShareTokenParam(attachment.externalLink, "motion");
  if (shareUrl) {
    return shareUrl;
  }

  return `${window.location.origin}/file/${attachment.name}/${encodeURIComponent(attachment.filename)}?motion=true`;
};

export const getAttachmentType = (attachment: Attachment) => {
  if (isImage(attachment.type)) {
    return "image/*";
  } else if (attachment.type.startsWith("video")) {
    return "video/*";
  } else if (attachment.type.startsWith("audio") && !isMidiFile(attachment.type)) {
    return "audio/*";
  } else if (attachment.type.startsWith("text")) {
    return "text/*";
  } else if (attachment.type.startsWith("application/epub+zip")) {
    return "application/epub+zip";
  } else if (attachment.type.startsWith("application/pdf")) {
    return "application/pdf";
  } else if (attachment.type.includes("word")) {
    return "application/msword";
  } else if (attachment.type.includes("excel")) {
    return "application/msexcel";
  } else if (attachment.type.startsWith("application/zip")) {
    return "application/zip";
  } else if (attachment.type.startsWith("application/x-java-archive")) {
    return "application/x-java-archive";
  } else {
    return "application/octet-stream";
  }
};

// isImage returns true if the given mime type is an image.
export const isImage = (t: string) => {
  // Don't show PSDs as images.
  return t.startsWith("image/") && !isPSD(t);
};

// isMidiFile returns true if the given mime type is a MIDI file.
export const isMidiFile = (mimeType: string): boolean => {
  return mimeType === "audio/midi" || mimeType === "audio/mid" || mimeType === "audio/x-midi" || mimeType === "application/x-midi";
};

const isPSD = (t: string) => {
  return t === "image/vnd.adobe.photoshop" || t === "image/x-photoshop" || t === "image/photoshop";
};

export const getAttachmentMotionGroupId = (attachment: Attachment): string | undefined => {
  return attachment.motionMedia?.groupId || undefined;
};

export const isAppleLivePhotoStill = (attachment: Attachment): boolean =>
  attachment.motionMedia?.family === MotionMediaFamily.APPLE_LIVE_PHOTO && attachment.motionMedia.role === MotionMediaRole.STILL;

export const isAppleLivePhotoVideo = (attachment: Attachment): boolean =>
  attachment.motionMedia?.family === MotionMediaFamily.APPLE_LIVE_PHOTO && attachment.motionMedia.role === MotionMediaRole.VIDEO;

export const isAndroidMotionContainer = (attachment: Attachment): boolean =>
  attachment.motionMedia?.family === MotionMediaFamily.ANDROID_MOTION_PHOTO &&
  attachment.motionMedia.role === MotionMediaRole.CONTAINER &&
  attachment.motionMedia.hasEmbeddedVideo;

export const isMotionAttachment = (attachment: Attachment): boolean =>
  isAppleLivePhotoStill(attachment) || isAppleLivePhotoVideo(attachment) || isAndroidMotionContainer(attachment);
