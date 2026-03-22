import { Page } from "@playwright/test";

export interface CanvasColorResult {
  totalPixels: number;
  coloredPixels: number;
  hasColors: boolean;
  sampleColors: Array<{ r: number; g: number; b: number; a: number }>;
}

export interface WebGLRenderResult {
  width: number;
  height: number;
  totalPixels: number;
  nonBlackPixels: number;
  hasContent: boolean;
}

/**
 * Validates that a 2D canvas has rendered colored pixels (non-transparent, non-black).
 * Used for LED preview canvases on dashboard and community pages.
 */
export async function assertCanvasHasColors(
  page: Page,
  selector: string
): Promise<CanvasColorResult> {
  return page.evaluate((sel: string) => {
    const canvas = document.querySelector(sel) as HTMLCanvasElement | null;
    if (!canvas) {
      return { totalPixels: 0, coloredPixels: 0, hasColors: false, sampleColors: [] };
    }
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return { totalPixels: 0, coloredPixels: 0, hasColors: false, sampleColors: [] };
    }
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    const data = imageData.data;
    const totalPixels = canvas.width * canvas.height;
    let coloredPixels = 0;
    const sampleColors: Array<{ r: number; g: number; b: number; a: number }> = [];

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i], g = data[i + 1], b = data[i + 2], a = data[i + 3];
      if (a > 0 && (r > 10 || g > 10 || b > 10)) {
        coloredPixels++;
        if (sampleColors.length < 5) {
          sampleColors.push({ r, g, b, a });
        }
      }
    }

    return { totalPixels, coloredPixels, hasColors: coloredPixels > 0, sampleColors };
  }, selector);
}

/**
 * Validates that a WebGL canvas (e.g., Three.js) has rendered content.
 * Uses toDataURL() since WebGL contexts can't use getContext('2d').
 */
export async function assertWebGLCanvasRendered(
  page: Page,
  containerSelector: string
): Promise<WebGLRenderResult> {
  return page.evaluate(async (sel: string) => {
    const container = document.querySelector(sel);
    const canvas = container?.querySelector("canvas") as HTMLCanvasElement | null;
    if (!canvas) {
      return { width: 0, height: 0, totalPixels: 0, nonBlackPixels: 0, hasContent: false };
    }

    const dataURL = canvas.toDataURL();
    const img = new Image();
    await new Promise<void>((resolve) => {
      img.onload = () => resolve();
      img.src = dataURL;
    });

    const tmpCanvas = document.createElement("canvas");
    tmpCanvas.width = img.width;
    tmpCanvas.height = img.height;
    const ctx = tmpCanvas.getContext("2d")!;
    ctx.drawImage(img, 0, 0);
    const imageData = ctx.getImageData(0, 0, tmpCanvas.width, tmpCanvas.height);
    const data = imageData.data;
    const totalPixels = tmpCanvas.width * tmpCanvas.height;
    let nonBlackPixels = 0;

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i], g = data[i + 1], b = data[i + 2];
      if (r > 10 || g > 10 || b > 10) {
        nonBlackPixels++;
      }
    }

    return {
      width: img.width,
      height: img.height,
      totalPixels,
      nonBlackPixels,
      hasContent: nonBlackPixels > totalPixels * 0.01,
    };
  }, containerSelector);
}
