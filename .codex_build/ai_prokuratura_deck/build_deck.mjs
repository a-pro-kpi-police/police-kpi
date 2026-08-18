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
  white: "#FFFFFF",
};
const FONT = "Arial";

const sourceNote =
  "[Sources]\n- User-provided pasted text: C:\\Users\\amanjohn\\.codex\\attachments\\385e71e9-8c7d-47ac-9073-4797e66188b3\\pasted-text.txt";

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

function addLine(slide, { x, y, w, h = 0, color = COLORS.ink, width = 1, name }) {
  return slide.shapes.add({
    geometry: "line",
    name,
    position: { left: x, top: y, width: w, height: h },
    fill: "none",
    line: { style: "solid", fill: color, width },
  });
}

function addText(
  slide,
  content,
  { x, y, w, h, size = 20, bold = false, color = COLORS.ink, align = "left", valign = "top", name, fit = "none" },
) {
  const shape = slide.shapes.add({
    geometry: "textbox",
    name,
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
  const {
    bulletCharacter = "•",
    marginLeft = 20,
    indent = -12,
    spaceAfter = 8,
    labelBold = false,
  } = options;
  return items.map((item) => {
    if (Array.isArray(item)) {
      return {
        bulletCharacter,
        marginLeft,
        indent,
        spaceAfter,
        runs: item,
      };
    }
    if (typeof item === "object") {
      return {
        bulletCharacter,
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
      bulletCharacter,
      marginLeft,
      indent,
      spaceAfter,
      runs: [{ run: item, textStyle: { bold: labelBold } }],
    };
  });
}

function notes(slide) {
  slide.speakerNotes.textFrame.setText(sourceNote);
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
  addText(slide, supertitle, { x: M, y: 44, w: 420, h: 36, size: 24, bold: true });
  addText(slide, title, { x: M, y: 178, w: 850, h: 260, size: 60, bold: true, fit: "shrinkText" });
  addText(slide, subtitle, { x: M, y: 478, w: 690, h: 82, size: 28, color: COLORS.muted });
  addLine(slide, { x: M, y: 612, w: 350, color: COLORS.accent, width: 4 });
  addText(slide, meta, { x: M, y: 636, w: 620, h: 32, size: 20, color: COLORS.muted });
  addBox(slide, { x: 1012, y: 92, w: 156, h: 156, fill: COLORS.accentLight });
  addBox(slide, { x: 1084, y: 164, w: 116, h: 116, fill: COLORS.light });
  notes(slide);
}

function sectionSlide(presentation, n, { title, lead, bullets: items, callout }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 910, h: 94, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 154, w: 1168, color: COLORS.rule });
  addText(slide, lead, { x: M, y: 194, w: 430, h: 270, size: 28, color: COLORS.ink, fit: "shrinkText" });
  addText(slide, bullets(items), { x: 560, y: 194, w: 610, h: 324, size: 21, color: COLORS.ink, fit: "shrinkText" });
  addBox(slide, { x: M, y: 548, w: 1110, h: 70, fill: COLORS.light });
  addText(slide, callout, { x: M + 22, y: 566, w: 1055, h: 42, size: 24, bold: true, fit: "shrinkText" });
  footer(slide, n);
  notes(slide);
}

function twoColumnSlide(presentation, n, { title, leftTitle, leftBody, rightTitle, rightBody, bottom }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1050, h: 88, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 150, w: 1168, color: COLORS.rule });
  addText(slide, leftTitle, { x: M, y: 198, w: 500, h: 38, size: 26, bold: true });
  addText(slide, leftBody, { x: M, y: 256, w: 500, h: 270, size: 22, color: COLORS.ink, fit: "shrinkText" });
  addLine(slide, { x: 612, y: 198, w: 0, h: 326, color: COLORS.rule });
  addText(slide, rightTitle, { x: 680, y: 198, w: 500, h: 38, size: 26, bold: true });
  addText(slide, rightBody, { x: 680, y: 256, w: 500, h: 270, size: 22, color: COLORS.ink, fit: "shrinkText" });
  if (bottom) {
    addBox(slide, { x: M, y: 562, w: 1110, h: 58, fill: COLORS.light });
    addText(slide, bottom, { x: M + 22, y: 578, w: 1058, h: 32, size: 23, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function threeColumnSlide(presentation, n, { title, columns, bottom }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1080, h: 92, size: 42, bold: true, fit: "shrinkText" });
  addLine(slide, { x: M, y: 150, w: 1168, color: COLORS.rule });
  const colW = 344;
  const gap = 54;
  columns.forEach((col, idx) => {
    const x = M + idx * (colW + gap);
    addText(slide, col.label, { x, y: 208, w: colW, h: 38, size: 26, bold: true, color: idx === 1 ? COLORS.accent : COLORS.ink });
    addText(slide, col.body, { x, y: 274, w: colW, h: 246, size: 20, fit: "shrinkText" });
    if (idx < 2) addLine(slide, { x: x + colW + 25, y: 208, w: 0, h: 316, color: COLORS.rule });
  });
  if (bottom) {
    addText(slide, bottom, { x: M, y: 568, w: 1060, h: 52, size: 25, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function processSlide(presentation, n, { title, steps, lead, bottom }) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, title, { x: M, y: 44, w: 1080, h: 92, size: 42, bold: true, fit: "shrinkText" });
  if (lead) addText(slide, lead, { x: M, y: 148, w: 1000, h: 42, size: 24, color: COLORS.muted, fit: "shrinkText" });
  const y = 326;
  const x0 = 80;
  const stepW = 190;
  const gap = 44;
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
    addText(slide, step, { x, y, w: stepW, h: 130, size: 21, bold: true, fit: "shrinkText" });
  });
  if (bottom) {
    addText(slide, bottom, { x: M, y: 548, w: 1090, h: 70, size: 26, bold: true, fit: "shrinkText" });
  }
  footer(slide, n);
  notes(slide);
}

function compareProcessSlide(presentation, n) {
  const slide = presentation.slides.add();
  slide.background.fill = COLORS.white;
  addText(slide, "Формат обучения начинается с рабочей задачи", { x: M, y: 44, w: 1100, h: 90, size: 42, bold: true, fit: "shrinkText" });
  addText(slide, "Вместо демонстрации инструмента участники решают знакомую рабочую задачу с помощью AI.", {
    x: M,
    y: 150,
    w: 890,
    h: 58,
    size: 25,
    color: COLORS.muted,
  });
  addText(slide, "Задача", { x: M, y: 246, w: 180, h: 34, size: 24, bold: true });
  addText(slide, "Изучить большой объем документов и подготовить аналитическую справку.", {
    x: 224,
    y: 246,
    w: 780,
    h: 42,
    size: 24,
    fit: "shrinkText",
  });
  addLine(slide, { x: M, y: 316, w: 1110, color: COLORS.rule });
  addText(slide, "Традиционный подход", { x: M, y: 360, w: 360, h: 34, size: 25, bold: true });
  addText(slide, "Документы → ручное чтение → поиск → сопоставление → справка", {
    x: M,
    y: 410,
    w: 1030,
    h: 36,
    size: 25,
    fit: "shrinkText",
  });
  addText(slide, "AI-подход", { x: M, y: 486, w: 240, h: 34, size: 25, bold: true, color: COLORS.accent });
  addText(slide, "Документы → AI-анализ → структура → взаимосвязи → черновик → проверка сотрудником", {
    x: M,
    y: 536,
    w: 1070,
    h: 38,
    size: 25,
    fit: "shrinkText",
  });
  addText(slide, "Финальное решение всегда остается за сотрудником.", {
    x: M,
    y: 610,
    w: 760,
    h: 34,
    size: 24,
    bold: true,
  });
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

async function main() {
  await fs.mkdir(TMP_DIR, { recursive: true });
  await fs.writeFile(path.join(TMP_DIR, "source-notes.txt"), sourceNote, "utf8");

  const presentation = Presentation.create({
    slideSize: { width: W, height: H },
  });

  titleSlide(presentation, {
    supertitle: "QAP Solutions",
    title: "Искусственный интеллект в прокуратуре",
    subtitle: "Обучение сотрудников и внедрение AI-решений в рабочие процессы",
    meta: "Атырау, 2026",
  });

  sectionSlide(presentation, 1, {
    title: "ИИ разгружает рутинную интеллектуальную работу",
    lead: "Главная задача — усилить сотрудника и вернуть время на содержательные решения.",
    bullets: [
      "анализ больших массивов документов",
      "структурирование информации",
      "поиск взаимосвязей и закономерностей",
      "подготовка аналитических материалов",
      "ускорение подготовки документов",
      "поддержка решений на основе данных",
    ],
    callout: "ИИ не заменяет сотрудника, а расширяет его возможности.",
  });

  sectionSlide(presentation, 2, {
    title: "Прокуратура ежедневно работает с большим объемом информации",
    lead: "Время часто уходит не на решение, а на поиск, чтение, сопоставление и подготовку данных.",
    bullets: [
      "нормативные и правовые документы",
      "обращения граждан и материалы проверок",
      "служебная переписка и аналитические справки",
      "отчеты, статистика, документы организаций и госорганов",
    ],
    callout: "AI позволяет перевести часть этой работы из ручного режима в автоматизированный.",
  });

  twoColumnSlide(presentation, 3, {
    title: "Проект строится на двух направлениях",
    leftTitle: "01 — AI Education",
    leftBody: "Практическое обучение сотрудников применению искусственного интеллекта на реальных рабочих задачах прокуратуры.",
    rightTitle: "02 — AI Implementation",
    rightBody: "Разработка и внедрение AI-инструментов непосредственно в рабочие процессы: документы, поиск, аналитика, отчеты.",
    bottom: "Формула: обучение → аудит процессов → пилот → измерение эффекта → масштабирование.",
  });

  sectionSlide(presentation, 4, {
    title: "Обучение привязано к реальным задачам сотрудников",
    lead: "Это не общий курс по ChatGPT, а практическая работа с типовыми процессами прокуратуры.",
    bullets: [
      "современная AI-грамотность",
      "эффективное взаимодействие с AI",
      "анализ документов и больших объемов информации",
      "подготовка аналитических материалов",
      "автоматизация рутинных задач",
      "проверка достоверности и безопасная работа с данными",
    ],
    callout: "Ценность обучения появляется там, где сотрудник сразу видит применение в своей работе.",
  });

  compareProcessSlide(presentation, 5);

  sectionSlide(presentation, 6, {
    title: "AI-помощник может стать единым окном к документам",
    lead: "Сотрудник работает с внутренними материалами через понятный интерфейс, задавая вопросы обычным языком.",
    bullets: [
      "поиск по документам",
      "краткое содержание",
      "ответы на вопросы",
      "сравнение документов",
      "извлечение нужной информации",
      "подготовка структурированных материалов",
      "ссылки на исходные документы",
    ],
    callout: "Результат остается проверяемым: ответ связан с первоисточниками.",
  });

  threeColumnSlide(presentation, 7, {
    title: "AI-анализ документов закрывает поиск, сравнение и классификацию",
    columns: [
      {
        label: "Поиск",
        body: bullets(["нужные документы", "конкретные факты", "упоминания", "связанные материалы"]),
      },
      {
        label: "Сравнение",
        body: bullets(["документы между собой", "версии документов", "показатели за периоды"]),
      },
      {
        label: "Анализ",
        body: bullets(["повторяющиеся ситуации", "классификация документов", "структура информации", "потенциальные несоответствия"]),
      },
    ],
    bottom: "Система помогает обработать массив, но выводы проверяет сотрудник.",
  });

  processSlide(presentation, 8, {
    title: "AI-поиск дает проверяемый ответ, а не просто текст",
    lead: "Пример вопроса: «Какие аналогичные случаи рассматривались ранее и какие нарушения были выявлены?»",
    steps: ["Находит документы", "Анализирует содержание", "Формирует ответ", "Показывает источники"],
    bottom: "Сотрудник получает ответ, который можно сверить по первоисточникам.",
  });

  sectionSlide(presentation, 9, {
    title: "AI-аналитика помогает увидеть проблемы в данных",
    lead: "ИИ применим не только к документам, но и к статистике, динамике и повторяющимся показателям.",
    bullets: [
      "анализ статистики и динамики",
      "поиск аномальных показателей",
      "выявление повторяющихся нарушений",
      "сравнение районов и подразделений",
      "выявление проблемных направлений",
      "автоматическое формирование аналитических отчетов",
    ],
    callout: "Данные → аналитика → выявление проблемы → управленческое решение.",
  });

  twoColumnSlide(presentation, 10, {
    title: "Безопасность должна быть встроена в контур внедрения",
    leftTitle: "Ограничение",
    leftBody: "Для государственных органов нельзя просто передавать рабочие документы в публичные AI-сервисы.",
    rightTitle: "Что нужно предусмотреть",
    rightBody: bullets([
      "разграничение доступа",
      "авторизацию пользователей",
      "журналирование действий",
      "защиту данных",
      "согласованную инфраструктуру",
      "корпоративный AI-контур",
      "проверку результатов человеком",
    ]),
    bottom: "AI помогает сотруднику, но не принимает юридически значимое решение самостоятельно.",
  });

  sectionSlide(presentation, 11, {
    title: "Начинать стоит с AI-аудита, а не с большой системы",
    lead: "Первый этап — изучить реальные процессы сотрудников и найти задачи с наибольшим потенциалом.",
    bullets: [
      "где тратится больше всего времени",
      "какие задачи повторяются",
      "где используется много документов",
      "какие процессы можно автоматизировать",
    ],
    callout: "Результат аудита — карта AI-возможностей прокуратуры.",
  });

  threeColumnSlide(presentation, 12, {
    title: "Пилот проверяет эффект на 2–3 процессах",
    columns: [
      { label: "Pilot №1", body: "AI-анализ документов." },
      { label: "Pilot №2", body: "AI-поиск по внутренней базе знаний." },
      { label: "Pilot №3", body: "AI-подготовка аналитических материалов." },
    ],
    bottom: "Для каждого пилота фиксируются KPI: время до/после, объем документов, точность, экономия рабочего времени.",
  });

  processSlide(presentation, 13, {
    title: "Масштабирование начинается после подтвержденного эффекта",
    steps: [
      "Подтвердить экономический и управленческий эффект",
      "Доработать решение",
      "Подключить новые подразделения",
      "Расширить AI-сценарии",
      "Сформировать корпоративный AI-контур",
    ],
    bottom: "Пилот превращается в устойчивую систему только после измеренного результата.",
  });

  threeColumnSlide(presentation, 14, {
    title: "Прокуратура получает эффект на трех уровнях",
    columns: [
      {
        label: "Сотрудники",
        body: bullets(["меньше рутины", "быстрый поиск информации", "удобная работа с документами", "рост цифровых компетенций"]),
      },
      {
        label: "Руководители",
        body: bullets(["оперативная аналитика", "прозрачность процессов", "быстрое получение информации", "контроль показателей"]),
      },
      {
        label: "Организация",
        body: "Рост производительности без пропорционального увеличения нагрузки на сотрудников.",
      },
    ],
    bottom: "Эффект должен быть заметен в ежедневной работе, управлении и организации процессов.",
  });

  twoColumnSlide(presentation, 15, {
    title: "QAP Solutions объединяет обучение, разработку и трансформацию",
    leftTitle: "Три компетенции",
    leftBody: bullets([
      { label: "AI Education:", body: "обучение работе с современными AI-инструментами" },
      { label: "AI Development:", body: "разработка решений под конкретные задачи" },
      { label: "Digital Transformation:", body: "автоматизация процессов и интеграция систем" },
    ]),
    rightTitle: "Подход",
    rightBody: "Не продавать AI ради AI. Сначала определить проблему, затем проверить AI на реальном процессе, измерить результат и только после этого масштабировать.",
    bottom: "Практический эффект важнее демонстрации технологии.",
  });

  numberedListSlide(presentation, 16, {
    title: "Следующий шаг — совместная AI-сессия",
    intro: "Встреча с представителями прокуратуры поможет быстро выбрать процессы для пилота.",
    items: [
      "Определить 10–15 наиболее трудоемких процессов.",
      "Выбрать процессы с максимальным потенциалом автоматизации.",
      "Определить 2–3 пилотных сценария.",
      "Сформировать техническую концепцию.",
      "Запустить пилот.",
    ],
    goal: "Цель — показать на реальных задачах конкретный эффект AI для прокуратуры Атырауской области.",
  });

  titleSlide(presentation, {
    supertitle: "Финальный тезис",
    title: "ИИ — это не замена сотруднику",
    subtitle: "Это инструмент, который помогает работать быстрее, глубже и эффективнее.",
    meta: "QAP Solutions",
  });

  const renderDir = path.join(TMP_DIR, "artifact-preview");
  await fs.mkdir(renderDir, { recursive: true });
  for (const [index, slide] of presentation.slides.items.entries()) {
    const stem = `slide-${String(index + 1).padStart(2, "0")}`;
    const png = await presentation.export({ slide, format: "png", scale: 1 });
    await fs.writeFile(path.join(renderDir, `${stem}.png`), new Uint8Array(await png.arrayBuffer()));
    const layout = await slide.export({ format: "layout" });
    await fs.writeFile(path.join(renderDir, `${stem}.layout.json`), await layout.text(), "utf8");
  }
  const montage = await presentation.export({ format: "webp", montage: true, scale: 1 });
  await fs.writeFile(path.join(TMP_DIR, "deck-montage.webp"), new Uint8Array(await montage.arrayBuffer()));

  const pptx = await PresentationFile.exportPptx(presentation);
  await pptx.save(FINAL_PPTX);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
