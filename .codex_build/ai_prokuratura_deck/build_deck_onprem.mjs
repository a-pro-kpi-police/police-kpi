import fs from "node:fs/promises";
import path from "node:path";
import { Presentation, PresentationFile } from "@oai/artifact-tool";

const TMP_DIR = process.env.TMP_DIR;
const FINAL_PPTX = process.env.FINAL_PPTX;

if (!TMP_DIR || !FINAL_PPTX) {
  throw new Error("TMP_DIR and FINAL_PPTX must be set.");
}

const W = 1280;
const H = 720;
const M = 56;
const COLORS = {
  ink: "#000000",
  muted: "#545B64",
  light: "#EDEDED",
  rule: "#B8BCC4",
  accent: "#3D8DFF",
  accentLight: "#D8EFFB",
  safe: "#DFF5EA",
  white: "#FFFFFF",
};
const FONT = "Arial";

const sourceNote =
  "[Sources]\n" +
  "- User-provided pasted text: C:\\Users\\amanjohn\\.codex\\attachments\\385e71e9-8c7d-47ac-9073-4797e66188b3\\pasted-text.txt\n" +
  "- User follow-up: adapt for non-technical audience; emphasize training, application development, and on-premise deployment.";

const visualAssetNote =
  sourceNote +
  "\n- Generated visual asset: .codex_build\\ai_prokuratura_deck\\assets\\ai-audit-onprem-visual.png";

function addBox(slide, { x, y, w, h, fill = COLORS.light, line = "none", name }) {
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

function addLine(slide, { x, y, w, h = 0, color = COLORS.ink, width = 1 }) {
  return slide.shapes.add({
    geometry: "line",
    position: { left: x, top: y, width: w, height: h },
    fill: "none",
    line: { style: "solid", fill: color, width },
  });
}

function addText(
  slide,
  content,
  { x, y, w, h, size = 20, bold = false, color = COLORS.ink, align = "left", valign = "top", fit = "none" },
) {
  const shape = slide.shapes.add({
    geometry: "textbox",
    position: { left: x, top: y, width: w, height: h },
    fill: "none",
    line: { style: "solid", fill: "none", width: 0 },
  });
  if (Array.isArray(content) && content.some((item) => typeof item === "object")) {
    shape.text.set(content);
  } else {
    shape.text = content;
  }
  shape.text.style = {
    fontSize: size,
    bold,
    color,
    alignment: align,
    verticalAlignment: valign,
    typeface: FONT,
    wrap: "square",
    autoFit: fit,
    insets: { top: 0, right: 0, bottom: 0, left: 0 },
  };
  return shape;
}

function bullets(items, options = {}) {
  const { marginLeft = 20, indent = -12, spaceAfter = 8 } = options;
  return items.map((item) => {
    if (typeof item === "object" && !Array.isArray(item)) {
      return {
        bulletCharacter: "•",
        marginLeft,
        indent,
        spaceAfter,
        runs: [
          { run: item.label, textStyle: { bold: true } },
          item.body ? ` ${item.body}` : "",
        ],
      };
    }
    return {
      bulletCharacter: "•",
      marginLeft,
      indent,
      spaceAfter,
      runs: [String(item)],
    };
  });
}

function notes(slide) {
  slide.speakerNotes.textFrame.setText(sourceNote);
}

function visualNotes(slide) {
  slide.speakerNotes.textFrame.setText(visualAssetNote);
}

function footer(slide, n) {
  addText(slide, "QAP Solutions", {
    x: M,
    y: 666,
    w: 240,
    h: 24,
    size: 14,
    color: COLORS.muted,
  });
  addText(slide, String(n).padStart(2, "0"), {
    x: 1180,
    y: 666,
    w: 44,
    h: 24,
    size: 14,
    color: COLORS.muted,
    align: "right",
  });
}

function titleSlide(presentation, { supertitle, title, subtitle, meta }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, supertitle, { x: M, y: 44, w: 520, h: 36, size: 24, bold: true });
  addText(slide, title, { x: M, y: 164, w: 880, h: 278, size: 56, bold: true, fit: "shrinkText" });
  addText(slide, subtitle, { x: M, y: 486, w: 760, h: 88, size: 28, color: COLORS.muted, fit: "shrinkText" });
  addLine(slide, { x: M, y: 612, w: 350, color: COLORS.accent, width: 4 });
  addText(slide, meta, { x: M, y: 636, w: 620, h: 32, size: 20, color: COLORS.muted });
  addBox(slide, { x: 1012, y: 92, w: 156, h: 156, fill: COLORS.accentLight });
  addBox(slide, { x: 1084, y: 164, w: 116, h: 116, fill: COLORS.light });
  notes(slide);
}

async function readImageBlob(imagePath) {
  const bytes = await fs.readFile(imagePath);
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
}

function sectionSlide(presentation, n, { title, lead, items, callout, calloutFill = COLORS.light }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1030, h: 94, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  addText(slide, lead, { x: M, y: 196, w: 440, h: 278, size: 28, fit: "shrinkText" });
  addText(slide, bullets(items), { x: 560, y: 196, w: 610, h: 318, size: 22, fit: "shrinkText" });
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: calloutFill });
  addText(slide, callout, { x: M + 22, y: 566, w: 1055, h: 42, size: 24, bold: true, fit: "shrinkText" });
  footer(slide, n);
  notes(slide);
}

function twoColumnSlide(presentation, n, { title, leftTitle, leftBody, rightTitle, rightBody, bottom, bottomFill = COLORS.light }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1100, h: 92, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  addText(slide, leftTitle, { x: M, y: 208, w: 500, h: 40, size: 27, bold: true });
  addText(slide, leftBody, { x: M, y: 278, w: 500, h: 250, size: 23, fit: "shrinkText" });
  addLine(slide, { x: 612, y: 208, w: 0, h: 318, color: COLORS.rule });
  addText(slide, rightTitle, { x: 680, y: 208, w: 500, h: 40, size: 27, bold: true, color: COLORS.accent });
  addText(slide, rightBody, { x: 680, y: 278, w: 500, h: 250, size: 23, fit: "shrinkText" });
  if (bottom) {
    addBox(slide, { x: M, y: 562, w: 1110, h: 58, fill: bottomFill });
    addText(slide, bottom, { x: M + 22, y: 578, w: 1058, h: 32, size: 23, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function threeColumnSlide(presentation, n, { title, columns, bottom }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1100, h: 92, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  const colW = 344;
  const gap = 54;
  columns.forEach((col, idx) => {
    const x = M + idx * (colW + gap);
    addText(slide, col.label, { x, y: 212, w: colW, h: 42, size: 27, bold: true, color: idx === 1 ? COLORS.accent : COLORS.ink });
    addText(slide, col.body, { x, y: 284, w: colW, h: 230, size: 21, fit: "shrinkText" });
    if (idx < 2) addLine(slide, { x: x + colW + 25, y: 212, w: 0, h: 314, color: COLORS.rule });
  });
  if (bottom) {
    addText(slide, bottom, { x: M, y: 568, w: 1080, h: 52, size: 25, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function processSlide(presentation, n, { title, lead, steps, bottom }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1110, h: 92, size: 42, bold: true, fit: "shrinkText" });
  if (lead) addText(slide, lead, { x: M, y: 148, w: 1100, h: 52, size: 24, color: COLORS.muted, fit: "shrinkText" });
  const y = 330;
  const x0 = 82;
  const stepW = steps.length === 5 ? 190 : 230;
  const gap = steps.length === 5 ? 44 : 64;
  addLine(slide, { x: x0 + 20, y: y - 28, w: 1080, color: COLORS.rule, width: 2 });
  steps.forEach((step, idx) => {
    const x = x0 + idx * (stepW + gap);
    addBox(slide, { x, y: y - 70, w: 44, h: 44, fill: idx === 0 ? COLORS.accent : COLORS.ink });
    addText(slide, String(idx + 1).padStart(2, "0"), {
      x: x + 8,
      y: y - 61,
      w: 28,
      h: 24,
      size: 16,
      bold: true,
      color: COLORS.white,
      align: "center",
    });
    addText(slide, step, { x, y, w: stepW, h: 136, size: 21, bold: true, fit: "shrinkText" });
  });
  if (bottom) {
    addText(slide, bottom, { x: M, y: 548, w: 1090, h: 70, size: 26, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function numberedListSlide(presentation, n, { title, intro, items, goal }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1100, h: 90, size: 42, bold: true, fit: "shrinkText" });
  addText(slide, intro, { x: M, y: 150, w: 1060, h: 48, size: 24, color: COLORS.muted, fit: "shrinkText" });
  const startY = 238;
  items.forEach((item, idx) => {
    const y = startY + idx * 64;
    addText(slide, String(idx + 1), { x: M, y, w: 44, h: 34, size: 25, bold: true, color: idx === 0 ? COLORS.accent : COLORS.ink });
    addText(slide, item, { x: 122, y, w: 980, h: 42, size: 23, fit: "shrinkText" });
    addLine(slide, { x: 122, y: y + 48, w: 980, color: COLORS.rule });
  });
  addText(slide, goal, { x: M, y: 582, w: 1040, h: 66, size: 25, bold: true, fit: "shrinkText" });
  footer(slide, n);
  notes(slide);
}

function visualAuditSlide(presentation, n, imageBytes) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, "Как выбирается первый пилот", {
    x: M,
    y: 44,
    w: 1040,
    h: 72,
    size: 42,
    bold: true,
    fit: "shrinkText",
  });
  addText(slide, "Смотрим на реальные документы и процессы, находим самые трудоемкие места и выбираем 2–3 сценария для проверки.", {
    x: M,
    y: 126,
    w: 1060,
    h: 48,
    size: 24,
    color: COLORS.muted,
    fit: "shrinkText",
  });
  slide.images.add({
    blob: imageBytes,
    contentType: "image/png",
    alt: "Визуальное объяснение выбора AI-пилота в закрытом on-premise контуре",
    fit: "contain",
    position: { left: 92, top: 198, width: 1096, height: 372 },
  });
  addText(slide, "Итог аудита — не большой проект сразу, а понятный пилот с измеримым эффектом и закрытым контуром данных.", {
    x: M,
    y: 604,
    w: 1090,
    h: 42,
    size: 24,
    bold: true,
    fit: "shrinkText",
  });
  footer(slide, n);
  visualNotes(slide);
}

function plainExplanationSlide(presentation, n) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, "ИИ в этом проекте — это рабочий помощник", { x: M, y: 44, w: 1060, h: 92, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  addText(slide, "Для сотрудников это не «сложная технология», а понятный инструмент рядом с документами.", {
    x: M,
    y: 206,
    w: 520,
    h: 150,
    size: 30,
    fit: "shrinkText",
  });
  addText(slide, bullets([
    { label: "Спросить:", body: "задать вопрос обычным языком." },
    { label: "Найти:", body: "быстро получить нужные документы и факты." },
    { label: "Собрать:", body: "подготовить черновик справки или анализа." },
    { label: "Проверить:", body: "сверить ответ по источникам и принять решение самому." },
  ]), {
    x: 660,
    y: 206,
    w: 520,
    h: 260,
    size: 23,
    fit: "shrinkText",
  });
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: COLORS.light });
  addText(slide, "Главный принцип: ИИ помогает готовить информацию, но не заменяет профессиональное решение сотрудника.", {
    x: M + 22,
    y: 566,
    w: 1055,
    h: 42,
    size: 24,
    bold: true,
    fit: "shrinkText",
  });
  footer(slide, n);
  notes(slide);
}

function securitySlide(presentation, n) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, "Разработка — только в закрытом контуре on-premise", {
    x: M,
    y: 44,
    w: 1080,
    h: 92,
    size: 42,
    bold: true,
    fit: "shrinkText",
  });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  addText(slide, "Все, что будет разработано, размещается внутри согласованной инфраструктуры прокуратуры.", {
    x: M,
    y: 210,
    w: 520,
    h: 160,
    size: 30,
    fit: "shrinkText",
  });
  addBox(slide, { x: 664, y: 208, w: 500, h: 255, fill: COLORS.safe, line: COLORS.rule });
  addText(slide, "Данные не уходят в интернет", { x: 696, y: 238, w: 430, h: 42, size: 28, bold: true });
  addText(slide, bullets([
    "рабочие документы не передаются в публичные AI-сервисы",
    "система работает внутри закрытого контура",
    "доступ выдается только авторизованным пользователям",
    "действия пользователей можно журналировать",
  ]), {
    x: 696,
    y: 310,
    w: 420,
    h: 124,
    size: 20,
    fit: "shrinkText",
  });
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: COLORS.safe });
  addText(slide, "Цель безопасности: ничего из внутренних материалов не должно «проскочить» в интернет.", {
    x: M + 22,
    y: 566,
    w: 1055,
    h: 42,
    size: 24,
    bold: true,
    fit: "shrinkText",
  });
  footer(slide, n);
  notes(slide);
}

async function writeBlob(filePath, blob) {
  await fs.writeFile(filePath, new Uint8Array(await blob.arrayBuffer()));
}

async function main() {
  await fs.mkdir(TMP_DIR, { recursive: true });
  await fs.writeFile(path.join(TMP_DIR, "source-notes-onprem.txt"), sourceNote, "utf8");
  const visualImagePath = path.join(TMP_DIR, "assets", "ai-audit-onprem-visual.png");
  const visualImageBytes = await readImageBlob(visualImagePath);

  const presentation = Presentation.create({
    slideSize: { width: W, height: H },
  });

  titleSlide(presentation, {
    supertitle: "QAP Solutions",
    title: "ИИ в прокуратуре: понятный и безопасный инструмент",
    subtitle: "Обучение сотрудников и разработка закрытого AI-приложения для рабочих процессов",
    meta: "Атырау, 2026",
  });

  plainExplanationSlide(presentation, 1);

  sectionSlide(presentation, 2, {
    title: "ИИ нужен там, где много ручной работы с информацией",
    lead: "Сотрудники тратят время на поиск, чтение, сопоставление и подготовку материалов.",
    items: [
      "нормативные и правовые документы",
      "обращения граждан и материалы проверок",
      "служебная переписка и аналитические справки",
      "отчеты, статистика, документы организаций и госорганов",
    ],
    callout: "ИИ берет на себя часть подготовки информации, а решение остается за сотрудником.",
  });

  twoColumnSlide(presentation, 3, {
    title: "Проект состоит из двух понятных направлений",
    leftTitle: "01 — Обучение сотрудников",
    leftBody: "Показываем, как использовать ИИ в реальных задачах: поиск, анализ документов, подготовка справок, проверка результата.",
    rightTitle: "02 — Разработка приложения",
    rightBody: "Создаем закрытый AI-помощник под процессы прокуратуры: документы, внутренняя база знаний, аналитика и отчеты.",
    bottom: "Сначала люди понимают инструмент, затем появляется приложение, встроенное в их работу.",
  });

  sectionSlide(presentation, 4, {
    title: "Обучение должно быть практическим, а не теоретическим",
    lead: "Участникам не нужно глубоко разбираться в технологиях, чтобы начать получать пользу.",
    items: [
      "что ИИ может и чего не может",
      "как задавать понятные вопросы",
      "как анализировать документы",
      "как готовить черновики аналитики",
      "как проверять ответы ИИ",
      "как работать с данными безопасно",
    ],
    callout: "Формат обучения: берем рабочую задачу сотрудника и решаем ее с помощью ИИ.",
  });

  twoColumnSlide(presentation, 5, {
    title: "Приложение становится единым окном для рабочих документов",
    leftTitle: "Как это выглядит для сотрудника",
    leftBody: "Сотрудник открывает внутренний AI-помощник, задает вопрос обычным языком и получает структурированный ответ.",
    rightTitle: "Что умеет система",
    rightBody: bullets([
      "искать по документам",
      "кратко пересказывать материалы",
      "сравнивать документы",
      "находить факты и связи",
      "готовить черновики справок",
      "показывать источники",
    ]),
    bottom: "Важно: AI-помощник работает с внутренними материалами, а не вместо сотрудника.",
  });

  processSlide(presentation, 6, {
    title: "Пример: сотрудник ищет похожие случаи",
    lead: "Вопрос можно задать обычным языком: «Какие аналогичные случаи уже рассматривались?»",
    steps: ["Находит документы", "Сравнивает содержание", "Собирает ответ", "Показывает источники"],
    bottom: "Ответ можно проверить по первоисточникам, поэтому результат остается контролируемым.",
  });

  threeColumnSlide(presentation, 7, {
    title: "AI-анализ документов закрывает три простые задачи",
    columns: [
      {
        label: "Найти",
        body: bullets(["нужные документы", "конкретные факты", "упоминания", "связанные материалы"]),
      },
      {
        label: "Сравнить",
        body: bullets(["документы между собой", "версии документов", "показатели за разные периоды"]),
      },
      {
        label: "Объяснить",
        body: bullets(["повторяющиеся ситуации", "структуру информации", "потенциальные несоответствия", "ключевые выводы"]),
      },
    ],
    bottom: "Система ускоряет подготовку, но выводы проверяет человек.",
  });

  securitySlide(presentation, 8);

  twoColumnSlide(presentation, 9, {
    title: "Безопасность закладывается на уровне архитектуры",
    leftTitle: "Что исключаем",
    leftBody: bullets([
      "передачу документов в публичные AI-сервисы",
      "хранение рабочих данных вне согласованного контура",
      "доступ без авторизации",
      "использование результата без проверки человеком",
    ]),
    rightTitle: "Что предусматриваем",
    rightBody: bullets([
      "on-premise размещение",
      "разграничение прав доступа",
      "журналирование действий",
      "контроль доступа к документам",
      "проверку источников и результатов",
    ]),
    bottom: "Безопасность — не отдельная опция, а обязательное условие внедрения.",
    bottomFill: COLORS.safe,
  });

  sectionSlide(presentation, 10, {
    title: "AI-аналитика помогает руководителям быстрее видеть картину",
    lead: "ИИ можно использовать не только для документов, но и для статистики, динамики и повторяющихся показателей.",
    items: [
      "анализ статистики и динамики",
      "поиск аномальных показателей",
      "выявление повторяющихся нарушений",
      "сравнение районов и подразделений",
      "подготовка аналитических отчетов",
    ],
    callout: "Данные → аналитика → выявление проблемы → управленческое решение.",
  });

  visualAuditSlide(presentation, 11, visualImageBytes);

  threeColumnSlide(presentation, 12, {
    title: "Пилот проверяет пользу на 2–3 рабочих сценариях",
    columns: [
      { label: "Сценарий 1", body: "AI-анализ большого массива документов." },
      { label: "Сценарий 2", body: "AI-поиск по внутренней базе знаний." },
      { label: "Сценарий 3", body: "AI-подготовка аналитической справки." },
    ],
    bottom: "Для каждого сценария фиксируем KPI: время до/после, точность, объем документов, экономия рабочего времени.",
  });

  processSlide(presentation, 13, {
    title: "Путь внедрения остается управляемым",
    steps: [
      "Обучить сотрудников",
      "Провести аудит процессов",
      "Выбрать пилот",
      "Разработать on-premise приложение",
      "Измерить эффект и масштабировать",
    ],
    bottom: "Масштабирование начинается только после понятного и измеренного результата.",
  });

  threeColumnSlide(presentation, 14, {
    title: "Что получает прокуратура",
    columns: [
      {
        label: "Сотрудники",
        body: bullets(["меньше рутинной работы", "быстрый поиск информации", "удобная работа с документами", "рост AI-грамотности"]),
      },
      {
        label: "Руководители",
        body: bullets(["оперативная аналитика", "прозрачность процессов", "быстрое получение справок", "контроль показателей"]),
      },
      {
        label: "Организация",
        body: bullets(["закрытый контур данных", "единый подход к AI", "рост производительности", "безопасное масштабирование"]),
      },
    ],
    bottom: "Эффект должен быть понятен людям, заметен в процессах и безопасен для данных.",
  });

  twoColumnSlide(presentation, 15, {
    title: "Почему QAP Solutions",
    leftTitle: "Мы соединяем две стороны проекта",
    leftBody: bullets([
      { label: "Обучение:", body: "помогаем людям понять инструмент и применять его без страха." },
      { label: "Разработка:", body: "создаем закрытое приложение под конкретные процессы." },
      { label: "Внедрение:", body: "двигаемся через аудит, пилот и измерение эффекта." },
    ]),
    rightTitle: "Наш подход",
    rightBody: "Не внедрять ИИ ради ИИ. Сначала найти реальную задачу, затем проверить решение на пилоте и только после этого расширять систему.",
    bottom: "Главное — практический эффект, безопасность и понятность для сотрудников.",
  });

  numberedListSlide(presentation, 16, {
    title: "Следующий шаг — совместная AI-сессия",
    intro: "На встрече можно быстро выбрать первые процессы для обучения, пилота и разработки приложения.",
    items: [
      "Определить 10–15 наиболее трудоемких процессов.",
      "Выбрать процессы с максимальным потенциалом автоматизации.",
      "Определить 2–3 пилотных сценария.",
      "Согласовать on-premise контур и требования безопасности.",
      "Запустить пилот.",
    ],
    goal: "Цель — показать на реальных задачах конкретный эффект ИИ для прокуратуры Атырауской области.",
  });

  titleSlide(presentation, {
    supertitle: "Финальный тезис",
    title: "ИИ — это безопасный инструмент в руках сотрудника",
    subtitle: "Обучаем людей, разрабатываем закрытое приложение и не выпускаем рабочие данные в интернет.",
    meta: "QAP Solutions",
  });

  const renderDir = path.join(TMP_DIR, "onprem-preview");
  await fs.mkdir(renderDir, { recursive: true });
  for (const [index, slide] of presentation.slides.items.entries()) {
    const stem = `slide-${String(index + 1).padStart(2, "0")}`;
    await writeBlob(path.join(renderDir, `${stem}.png`), await presentation.export({ slide, format: "png", scale: 1 }));
    const layout = await slide.export({ format: "layout" });
    await fs.writeFile(path.join(renderDir, `${stem}.layout.json`), await layout.text(), "utf8");
  }
  await writeBlob(path.join(TMP_DIR, "onprem-montage.webp"), await presentation.export({ format: "webp", montage: true, scale: 1 }));

  const pptx = await PresentationFile.exportPptx(presentation);
  await pptx.save(FINAL_PPTX);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
