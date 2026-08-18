import fs from "node:fs/promises";
import path from "node:path";
import { FileBlob, PresentationFile } from "@oai/artifact-tool";

const TMP_DIR = process.env.TMP_DIR;
const SOURCE_PPTX = process.env.SOURCE_PPTX;
const FINAL_PPTX = process.env.FINAL_PPTX;
const SECONDARY_PPTX = process.env.SECONDARY_PPTX;

if (!TMP_DIR || !SOURCE_PPTX || !FINAL_PPTX) {
  throw new Error("TMP_DIR, SOURCE_PPTX and FINAL_PPTX must be set.");
}

const COLORS = {
  ink: "#000000",
  muted: "#545B64",
  light: "#EDEDED",
  rule: "#B8BCC4",
  accent: "#3D8DFF",
  safe: "#DFF5EA",
  white: "#FFFFFF",
};

const FONT = "Arial";
const M = 56;

const notesText =
  "[Sources]\n" +
  "- User-provided pasted text: C:\\Users\\amanjohn\\.codex\\attachments\\385e71e9-8c7d-47ac-9073-4797e66188b3\\pasted-text.txt\n" +
  "- User follow-up: adapt for non-technical audience; emphasize training, application development, and on-premise deployment.\n" +
  "- Generated visual asset: .codex_build\\ai_prokuratura_deck\\assets\\training-visual.png\n" +
  "- Generated visual asset: .codex_build\\ai_prokuratura_deck\\assets\\onprem-app-visual.png\n" +
  "- Generated visual asset: .codex_build\\ai_prokuratura_deck\\assets\\ai-audit-onprem-visual.png";

async function readImageBlob(imagePath) {
  const bytes = await fs.readFile(imagePath);
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
}

function addBox(slide, { x, y, w, h, fill = COLORS.white, line = "none", name }) {
  return slide.shapes.add({
    geometry: "rect",
    name,
    position: { left: x, top: y, width: w, height: h },
    fill,
    line:
      line === "none"
        ? { style: "solid", fill: "none", width: 0 }
        : { style: "solid", fill: line, width: 1 },
  });
}

function addText(
  slide,
  content,
  { x, y, w, h, size = 20, bold = false, color = COLORS.ink, align = "left", fit = "none" },
) {
  const shape = slide.shapes.add({
    geometry: "textbox",
    position: { left: x, top: y, width: w, height: h },
    fill: "none",
    line: { style: "solid", fill: "none", width: 0 },
  });
  if (Array.isArray(content)) {
    shape.text.set(content);
  } else {
    shape.text = content;
  }
  shape.text.style = {
    fontSize: size,
    bold,
    color,
    alignment: align,
    typeface: FONT,
    wrap: "square",
    autoFit: fit,
    insets: { top: 0, right: 0, bottom: 0, left: 0 },
  };
  return shape;
}

function bullets(items) {
  return items.map((item) => ({
    bulletCharacter: "•",
    marginLeft: 20,
    indent: -12,
    spaceAfter: 7,
    runs: [item],
  }));
}

function addVisualFrame(slide, imageBytes, alt, position) {
  slide.images.add({
    blob: imageBytes,
    contentType: "image/png",
    alt,
    fit: "contain",
    position,
  });
}

function replaceTrainingSlide(slide, imageBytes) {
  addBox(slide, { x: M, y: 178, w: 1112, h: 472, fill: COLORS.white });
  addVisualFrame(slide, imageBytes, "Практическое обучение сотрудников работе с AI-инструментом", {
    left: M,
    top: 194,
    width: 568,
    height: 320,
  });
  addText(slide, "Обучение показывает ИИ как рабочий инструмент", {
    x: 660,
    y: 194,
    w: 500,
    h: 78,
    size: 30,
    bold: true,
    fit: "shrinkText",
  });
  addText(
    slide,
    bullets([
      "задаем вопросы обычным языком",
      "ищем нужные документы и факты",
      "готовим черновики справок",
      "проверяем ответы по источникам",
    ]),
    { x: 680, y: 292, w: 468, h: 150, size: 23, fit: "shrinkText" },
  );
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: COLORS.light });
  addText(slide, "Формат обучения: берем реальную рабочую задачу и решаем ее вместе с ИИ.", {
    x: M + 22,
    y: 566,
    w: 1055,
    h: 42,
    size: 24,
    bold: true,
    fit: "shrinkText",
  });
  slide.speakerNotes.textFrame.setText(notesText);
}

function replaceApplicationSlide(slide, imageBytes) {
  addBox(slide, { x: M, y: 178, w: 1112, h: 472, fill: COLORS.white });
  addText(slide, "Приложение работает внутри инфраструктуры прокуратуры", {
    x: M,
    y: 198,
    w: 492,
    h: 90,
    size: 30,
    bold: true,
    fit: "shrinkText",
  });
  addText(
    slide,
    "Сотрудник открывает внутренний AI-помощник, задает вопрос и получает ответ по рабочим материалам. Документы остаются внутри закрытого контура.",
    { x: M, y: 314, w: 492, h: 132, size: 24, color: COLORS.muted, fit: "shrinkText" },
  );
  addVisualFrame(slide, imageBytes, "On-premise AI-приложение в закрытом контуре без передачи данных в интернет", {
    left: 584,
    top: 194,
    width: 590,
    height: 332,
  });
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: COLORS.safe });
  addText(slide, "Главная мысль: ничего из внутренних материалов не должно уходить в интернет.", {
    x: M + 22,
    y: 566,
    w: 1055,
    h: 42,
    size: 24,
    bold: true,
    fit: "shrinkText",
  });
  slide.speakerNotes.textFrame.setText(notesText);
}

async function main() {
  const presentation = await PresentationFile.importPptx(await FileBlob.load(SOURCE_PPTX));

  const trainingImage = await readImageBlob(path.join(TMP_DIR, "assets", "training-visual.png"));
  const onpremImage = await readImageBlob(path.join(TMP_DIR, "assets", "onprem-app-visual.png"));

  replaceTrainingSlide(presentation.slides.items[4], trainingImage);
  replaceApplicationSlide(presentation.slides.items[5], onpremImage);
  presentation.slides.items[11].speakerNotes.textFrame.setText(notesText);

  const pptx = await PresentationFile.exportPptx(presentation);
  await pptx.save(FINAL_PPTX);
  if (SECONDARY_PPTX) {
    await pptx.save(SECONDARY_PPTX);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
