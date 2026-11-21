import { useState } from "react";
import { FiDownload } from "react-icons/fi";
import { resourceContentUrl } from "../api/client";
import { useTranslate } from "../hooks/useTranslate";
import type { Resource } from "../types/resource";
import { fileCategory } from "../util/resources";
import { ResourceIcon } from "./ResourceIcon";
import { TextFileEditor } from "./TextFileEditor";
import { Button } from "./Button";

type FilePreviewProps = {
    resource: Resource;
    onDownload: () => void;
};

// Shows a file the way its type allows: an editor for text, a viewer for
// images, PDFs, audio and video, and a download for the rest.
export function FilePreview({ resource, onDownload }: FilePreviewProps) {
    const t = useTranslate();
    const category = fileCategory(resource);
    const url = resourceContentUrl(resource);
    const readOnly = resource.trashedAt !== null;

    switch (category) {
        case "markdown":
        case "text":
            return (
                <TextFileEditor
                    key={resource.id}
                    resource={resource}
                    isMarkdown={category === "markdown"}
                    readOnly={readOnly}
                />
            );
        case "image":
            return <ImagePreview key={resource.id} url={url} name={resource.name} />;
        case "pdf":
            return <iframe src={url} title={resource.name} className="min-h-0 w-full flex-1 border-0" />;
        case "audio":
            return (
                <div className="flex flex-1 items-center justify-center p-8">
                    <audio controls src={url} className="w-full max-w-lg" />
                </div>
            );
        case "video":
            return (
                <div className="flex min-h-0 flex-1 items-center justify-center bg-black/90 p-4">
                    <video controls src={url} className="max-h-full max-w-full" />
                </div>
            );
        default:
            return (
                <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
                    <ResourceIcon resource={resource} className="text-7xl" />
                    <p className="text-sm text-gray-500">{t("resources.preview.unavailable")}</p>
                    <Button variant="primary" size="lg"
                        type="button"
                        onClick={onDownload}
                    >
                        <FiDownload aria-hidden />
                        {t("resources.download")}
                    </Button>
                </div>
            );
    }
}

function ImagePreview({ url, name }: { url: string; name: string }) {
    const t = useTranslate();
    const [isActualSize, setIsActualSize] = useState(false);
    return (
        <div className="flex min-h-0 flex-1 overflow-auto bg-[repeating-conic-gradient(#f3f4f6_0_25%,#fff_0_50%)] bg-[length:20px_20px] p-4">
            <button
                type="button"
                onClick={() => setIsActualSize(!isActualSize)}
                title={t(isActualSize ? "resources.preview.fit" : "resources.preview.actualSize")}
                className={`m-auto ${isActualSize ? "cursor-zoom-out" : "cursor-zoom-in"}`}
            >
                <img
                    src={url}
                    alt={name}
                    className={isActualSize ? "max-w-none" : "max-h-[calc(100vh-14rem)] max-w-full object-contain"}
                />
            </button>
        </div>
    );
}
