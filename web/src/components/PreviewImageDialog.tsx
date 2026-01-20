import { X } from "lucide-react";
import { MouseEvent, useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  imgUrls: string[];
  initialIndex?: number;
}

const MIN_SCALE = 0.1;
const MAX_SCALE = 5;
const SCALE_STEP = 0.25;

function PreviewImageDialog({ open, onOpenChange, imgUrls, initialIndex = 0 }: Props) {
  const [currentIndex, setCurrentIndex] = useState(initialIndex);
  const [scale, setScale] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const dragStartRef = useRef({ x: 0, y: 0, posX: 0, posY: 0 });
  const containerRef = useRef<HTMLDivElement | null>(null);

  const zoomIn = useCallback(() => setScale((prev) => Math.min(prev + SCALE_STEP, MAX_SCALE)), []);
  const zoomOut = useCallback(() => setScale((prev) => Math.max(prev - SCALE_STEP, MIN_SCALE)), []);
  const resetZoom = useCallback(() => {
    setScale(1);
    setPosition({ x: 0, y: 0 });
  }, []);

  // Callback ref to handle wheel zoom - ensures event listener is added when DOM mounts
  const setContainerRef = useCallback(
    (node: HTMLDivElement | null) => {
      const handleWheel = (event: WheelEvent) => {
        event.preventDefault();
        if (event.deltaY < 0) {
          zoomIn();
        } else {
          zoomOut();
        }
      };

      // Clean up old node
      if (containerRef.current) {
        containerRef.current.removeEventListener("wheel", handleWheel);
      }

      containerRef.current = node;

      // Bind to new node
      if (node) {
        node.addEventListener("wheel", handleWheel, { passive: false });
      }
    },
    [zoomIn, zoomOut],
  );

  // Update current index and reset zoom/position when initialIndex prop changes or dialog opens
  useEffect(() => {
    if (open) {
      setCurrentIndex(initialIndex);
      setScale(1);
      setPosition({ x: 0, y: 0 });
    }
  }, [initialIndex, open]);

  // Handle mouse drag for panning
  useEffect(() => {
    if (!open) return;

    const handleMouseMove = (event: globalThis.MouseEvent) => {
      if (!isDragging) return;
      const deltaX = event.clientX - dragStartRef.current.x;
      const deltaY = event.clientY - dragStartRef.current.y;
      setPosition({
        x: dragStartRef.current.posX + deltaX,
        y: dragStartRef.current.posY + deltaY,
      });
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [open, isDragging]);

  const handleImageMouseDown = (event: MouseEvent<HTMLImageElement>) => {
    event.preventDefault();
    setIsDragging(true);
    dragStartRef.current = {
      x: event.clientX,
      y: event.clientY,
      posX: position.x,
      posY: position.y,
    };
  };

  // Handle keyboard navigation and zoom
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!open) return;

      const isMod = event.metaKey || event.ctrlKey;

      if (event.key === "Escape") {
        onOpenChange(false);
        return;
      }

      if (isMod) {
        switch (event.key) {
          case "+":
          case "=":
            event.preventDefault();
            zoomIn();
            break;
          case "-":
            event.preventDefault();
            zoomOut();
            break;
          case "0":
            event.preventDefault();
            resetZoom();
            break;
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onOpenChange, zoomIn, zoomOut, resetZoom]);

  const handleClose = () => {
    onOpenChange(false);
  };

  const handleBackdropClick = (event: MouseEvent<HTMLDivElement>) => {
    // Only close if clicking on backdrop (not image) and not dragging
    if (event.target === event.currentTarget && !isDragging) {
      handleClose();
    }
  };

  // Return early if no images provided
  if (!imgUrls.length) return null;

  // Ensure currentIndex is within bounds
  const safeIndex = Math.max(0, Math.min(currentIndex, imgUrls.length - 1));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="!w-[100vw] !h-[100vh] !max-w-[100vw] !max-h-[100vw] p-0 border-0 shadow-none bg-transparent [&>button]:hidden"
        aria-describedby="image-preview-description"
      >
        {/* Close button */}
        <div className="fixed top-4 right-4 z-50">
          <Button
            onClick={handleClose}
            variant="secondary"
            size="icon"
            className="rounded-full bg-popover/20 hover:bg-popover/30 border-border/20 backdrop-blur-sm"
            aria-label="Close image preview"
          >
            <X className="h-4 w-4 text-popover-foreground" />
          </Button>
        </div>

        {/* Image container */}
        <div ref={setContainerRef} className="w-full h-full flex items-center justify-center p-4 sm:p-8 overflow-auto" onClick={handleBackdropClick}>
          <img
            src={imgUrls[safeIndex]}
            alt={`Preview image ${safeIndex + 1} of ${imgUrls.length}`}
            className={`max-w-full max-h-full object-contain select-none ${isDragging ? "" : "transition-transform"} ${isDragging ? "cursor-grabbing" : "cursor-grab"}`}
            style={{ transform: `translate(${position.x}px, ${position.y}px) scale(${scale})` }}
            draggable={false}
            loading="eager"
            decoding="async"
            onMouseDown={handleImageMouseDown}
          />
        </div>

        {/* Screen reader description */}
        <div id="image-preview-description" className="sr-only">
          Image preview dialog. Press Escape to close or click outside the image.
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default PreviewImageDialog;
