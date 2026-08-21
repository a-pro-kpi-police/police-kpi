from __future__ import annotations

import argparse
import csv
import math
from dataclasses import dataclass
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.pagesizes import A4
from reportlab.lib.units import mm
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.pdfgen import canvas


@dataclass(frozen=True)
class Account:
    number: int
    employee_number: str
    full_name: str
    username: str
    password: str
    notes: str


def register_fonts() -> tuple[str, str]:
    regular = Path(r"C:\Windows\Fonts\arial.ttf")
    bold = Path(r"C:\Windows\Fonts\arialbd.ttf")
    if not regular.exists() or not bold.exists():
        raise FileNotFoundError("Arial fonts were not found in C:\\Windows\\Fonts")
    pdfmetrics.registerFont(TTFont("DocRegular", str(regular)))
    pdfmetrics.registerFont(TTFont("DocBold", str(bold)))
    return "DocRegular", "DocBold"


def read_accounts(csv_path: Path) -> list[Account]:
    with csv_path.open("r", encoding="utf-8-sig", newline="") as f:
        rows = list(csv.DictReader(f))

    accounts: list[Account] = []
    for idx, row in enumerate(rows, start=1):
        account = Account(
            number=idx,
            employee_number=(row.get("employee_number") or "").strip(),
            full_name=(row.get("full_name") or "").strip(),
            username=(row.get("username") or "").strip(),
            password=(row.get("password") or "").strip(),
            notes=(row.get("notes") or "").strip(),
        )
        if not account.full_name or not account.username or not account.password:
            raise ValueError(f"Missing required field in row {idx}")
        accounts.append(account)

    usernames = [a.username for a in accounts]
    if len(usernames) != len(set(usernames)):
        raise ValueError("Duplicate usernames found")
    return accounts


def draw_wrapped(
    c: canvas.Canvas,
    text: str,
    x: float,
    y: float,
    width: float,
    font: str,
    size: float,
    leading: float,
    color=colors.black,
    max_lines: int | None = None,
) -> float:
    c.setFillColor(color)
    c.setFont(font, size)
    words = text.split()
    lines: list[str] = []
    current = ""
    for word in words:
        candidate = word if not current else f"{current} {word}"
        if c.stringWidth(candidate, font, size) <= width:
            current = candidate
        else:
            if current:
                lines.append(current)
            current = word
    if current:
        lines.append(current)

    if max_lines is not None and len(lines) > max_lines:
        lines = lines[:max_lines]
        while lines[-1] and c.stringWidth(lines[-1] + "...", font, size) > width:
            lines[-1] = lines[-1][:-1]
        lines[-1] = lines[-1] + "..."

    for line in lines:
        c.drawString(x, y, line)
        y -= leading
    return y


def field_box(
    c: canvas.Canvas,
    label: str,
    value: str,
    x: float,
    y: float,
    width: float,
    height: float,
    regular_font: str,
    bold_font: str,
    label_width: float,
    value_size: float,
) -> None:
    c.setStrokeColor(colors.HexColor("#B7C0CC"))
    c.setLineWidth(0.6)
    c.roundRect(x, y - height, width, height, 3.5, stroke=1, fill=0)
    c.setFillColor(colors.HexColor("#56616F"))
    c.setFont(regular_font, 7.5)
    c.drawString(x + 3.2 * mm, y - 6.1 * mm, label)
    c.setFillColor(colors.black)
    c.setFont(bold_font, value_size)
    c.drawString(x + label_width, y - 6.4 * mm, value)


def draw_card(
    c: canvas.Canvas,
    account: Account,
    system_url: str,
    x: float,
    y: float,
    w: float,
    h: float,
    regular_font: str,
    bold_font: str,
    compact: bool,
) -> None:
    pad_x = 8 * mm if compact else 15 * mm
    pad_y = 7 * mm if compact else 13 * mm
    inner_x = x + pad_x
    top = y + h - pad_y
    inner_w = w - 2 * pad_x

    c.setStrokeColor(colors.HexColor("#303640"))
    c.setLineWidth(0.9)
    c.roundRect(x + 2.5 * mm, y + 2.5 * mm, w - 5 * mm, h - 5 * mm, 5, stroke=1, fill=0)

    c.setFillColor(colors.HexColor("#233041"))
    c.setFont(bold_font, 8 if compact else 10)
    c.drawString(inner_x, top, "KPI «ОТДЕЛ ПОЛИЦИИ»")
    c.setFillColor(colors.HexColor("#6B7280"))
    c.setFont(regular_font, 7.2 if compact else 8.5)
    c.drawRightString(x + w - pad_x, top, f"{account.number:03d}")

    y_cursor = top - (6.7 * mm if compact else 9.5 * mm)
    name_size = 9.2 if compact else 15
    y_cursor = draw_wrapped(
        c,
        account.full_name,
        inner_x,
        y_cursor,
        inner_w,
        bold_font,
        name_size,
        4.5 * mm if compact else 7.1 * mm,
        max_lines=2,
    )

    c.setFillColor(colors.HexColor("#56616F"))
    c.setFont(regular_font, 7.3 if compact else 9)
    c.drawString(inner_x, y_cursor - (1.0 * mm if compact else 1.5 * mm), f"Табельный номер: {account.employee_number or '-'}")

    address_y = y_cursor - (6.8 * mm if compact else 10.8 * mm)
    c.setFillColor(colors.HexColor("#56616F"))
    c.setFont(regular_font, 7.4 if compact else 9)
    c.drawString(inner_x, address_y, "Адрес системы:")
    c.setFillColor(colors.black)
    c.setFont(bold_font, 8 if compact else 11)
    c.drawString(inner_x + (24 * mm if compact else 31 * mm), address_y, system_url)

    box_h = 9.0 * mm if compact else 13 * mm
    field_gap = 2.2 * mm if compact else 4 * mm
    login_y = address_y - (5.2 * mm if compact else 8 * mm)
    label_width = 19 * mm if compact else 31 * mm
    field_box(c, "Логин", account.username, inner_x, login_y, inner_w, box_h, regular_font, bold_font, label_width, 10.5 if compact else 16)
    password_y = login_y - box_h - field_gap
    field_box(c, "Пароль", account.password, inner_x, password_y, inner_w, box_h, regular_font, bold_font, label_width, 10.5 if compact else 16)

    instruction_y = password_y - box_h - (4.2 * mm if compact else 8 * mm)
    c.setFillColor(colors.HexColor("#233041"))
    c.setFont(bold_font, 7.6 if compact else 10.5)
    c.drawString(inner_x, instruction_y, "Короткая инструкция")
    instruction_y -= 4.2 * mm if compact else 6.3 * mm

    instructions = [
        f"1. Откройте {system_url} в браузере.",
        "2. Введите логин и пароль точно как написано.",
        "3. Проверьте, что открылись ваши ФИО и участок.",
        "4. Записи KPI: добавляйте показатели, дату, количество и файл.",
        "5. Целевые KPI, уведомления и мессенджер смотрите в меню.",
    ]
    if compact:
        instructions = [
            f"1. Откройте {system_url} в браузере.",
            "2. Введите логин и пароль точно как написано.",
            "3. Проверьте ФИО и участок.",
            "4. В «Записи KPI» добавляйте показатели и файлы.",
            "5. Цели, уведомления и сообщения смотрите в меню.",
        ]

    for line in instructions:
        instruction_y = draw_wrapped(
            c,
            line,
            inner_x,
            instruction_y,
            inner_w,
            regular_font,
            6.5 if compact else 9.2,
            3.3 * mm if compact else 5.2 * mm,
            max_lines=2,
        )

    footer_y = y + (8 * mm if compact else 15 * mm)
    c.setStrokeColor(colors.HexColor("#D4DAE2"))
    c.line(inner_x, footer_y + (5 * mm if compact else 7 * mm), x + w - pad_x, footer_y + (5 * mm if compact else 7 * mm))
    c.setFillColor(colors.HexColor("#1F2937"))
    c.setFont(bold_font, 6.4 if compact else 8.7)
    c.drawString(inner_x, footer_y + (1.7 * mm if compact else 2.6 * mm), "Важно: пароль только для вас. Не передавайте его другим.")
    c.setFillColor(colors.HexColor("#6B7280"))
    c.setFont(regular_font, 5.6 if compact else 7.5)
    c.drawString(inner_x, footer_y - (1.8 * mm if compact else 2.2 * mm), "Если не получается войти или данные неверные, обратитесь к администратору.")


def draw_cut_guides(c: canvas.Canvas, page_w: float, page_h: float) -> None:
    c.setStrokeColor(colors.HexColor("#AAB3BF"))
    c.setDash(2, 3)
    c.setLineWidth(0.45)
    c.line(page_w / 2, 7 * mm, page_w / 2, page_h - 7 * mm)
    c.line(7 * mm, page_h / 2, page_w - 7 * mm, page_h / 2)
    c.setDash()


def build_four_per_page(accounts: list[Account], system_url: str, output_path: Path, regular_font: str, bold_font: str) -> None:
    c = canvas.Canvas(str(output_path), pagesize=A4)
    page_w, page_h = A4
    card_w = page_w / 2
    card_h = page_h / 2
    positions = [
        (0, page_h / 2),
        (page_w / 2, page_h / 2),
        (0, 0),
        (page_w / 2, 0),
    ]

    for page_start in range(0, len(accounts), 4):
        draw_cut_guides(c, page_w, page_h)
        for pos, account in zip(positions, accounts[page_start : page_start + 4]):
            draw_card(c, account, system_url, pos[0], pos[1], card_w, card_h, regular_font, bold_font, compact=True)
        c.showPage()
    c.save()


def build_one_per_page(accounts: list[Account], system_url: str, output_path: Path, regular_font: str, bold_font: str) -> None:
    c = canvas.Canvas(str(output_path), pagesize=A4)
    page_w, page_h = A4
    card_margin_x = 22 * mm
    card_margin_y = 28 * mm
    for account in accounts:
        draw_card(
            c,
            account,
            system_url,
            card_margin_x,
            card_margin_y,
            page_w - 2 * card_margin_x,
            page_h - 2 * card_margin_y,
            regular_font,
            bold_font,
            compact=False,
        )
        c.showPage()
    c.save()


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate printable account cards for DPS employees.")
    parser.add_argument("--csv", required=True, type=Path)
    parser.add_argument("--out-dir", required=True, type=Path)
    parser.add_argument("--system-url", default="atyraudp.kz")
    args = parser.parse_args()

    accounts = read_accounts(args.csv)
    args.out_dir.mkdir(parents=True, exist_ok=True)
    regular_font, bold_font = register_fonts()

    four_pdf = args.out_dir / "dps_account_cards_4_per_page.pdf"
    single_pdf = args.out_dir / "dps_account_cards_1_per_page.pdf"
    build_four_per_page(accounts, args.system_url, four_pdf, regular_font, bold_font)
    build_one_per_page(accounts, args.system_url, single_pdf, regular_font, bold_font)

    print(f"accounts={len(accounts)}")
    print(f"four_per_page={four_pdf}")
    print(f"four_per_page_pages={math.ceil(len(accounts) / 4)}")
    print(f"one_per_page={single_pdf}")
    print(f"one_per_page_pages={len(accounts)}")


if __name__ == "__main__":
    main()
