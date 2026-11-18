// Prepares an uploaded picture for use as the app background: downscaled so a
// 20 MB camera photo becomes a few hundred KB, and re-encoded as WebP.

const MAX_WIDTH = 2560;
const MAX_HEIGHT = 1600;
const QUALITY = 0.82;

export type PreparedBackground = {
  blob: Blob;
  // Average color as #rrggbb, shown before the image has loaded.
  tint: string;
};

export class UnsupportedImageError extends Error {}

function toHex(value: number): string {
  return value.toString(16).padStart(2, "0");
}

export async function prepareBackground(file: File): Promise<PreparedBackground> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file);
  } catch {
    throw new UnsupportedImageError();
  }
  try {
    const scale = Math.min(1, MAX_WIDTH / bitmap.width, MAX_HEIGHT / bitmap.height);
    const width = Math.max(1, Math.round(bitmap.width * scale));
    const height = Math.max(1, Math.round(bitmap.height * scale));

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) throw new UnsupportedImageError();
    context.imageSmoothingQuality = "high";
    context.drawImage(bitmap, 0, 0, width, height);

    const blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, "image/webp", QUALITY),
    );
    if (!blob) throw new UnsupportedImageError();

    return { blob, tint: averageColor(canvas) };
  } finally {
    bitmap.close();
  }
}

// Averages a small downscaled copy: drawing straight into one pixel would
// sample only the middle of the picture.
function averageColor(source: HTMLCanvasElement): string {
  const size = 64;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const context = canvas.getContext("2d");
  if (!context) return "#808080";
  context.imageSmoothingQuality = "high";
  context.drawImage(source, 0, 0, size, size);
  const data = context.getImageData(0, 0, size, size).data;
  let r = 0;
  let g = 0;
  let b = 0;
  for (let i = 0; i < data.length; i += 4) {
    r += data[i];
    g += data[i + 1];
    b += data[i + 2];
  }
  const pixels = data.length / 4;
  return `#${toHex(Math.round(r / pixels))}${toHex(Math.round(g / pixels))}${toHex(Math.round(b / pixels))}`;
}
