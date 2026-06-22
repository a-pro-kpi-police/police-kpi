"""Начальное наполнение базы: участки, каталог KPI, пользователи и примеры записей."""
import random
from datetime import date, timedelta

from database import SessionLocal, engine, Base
from models import District, User, KpiCatalog, KpiEntry
from auth import hash_password

# Всего сотрудников с ролью officer в демо-базе (начальник считается отдельно).
TARGET_OFFICERS = 20
TARGET_KPI_ENTRIES = 1000

DISTRICTS = [
    ("Центральный участок", 0.9),
    ("Участок №1", 1.0),
    ("Участок №2", 1.1),
    ("Участок №3", 1.2),
    ("Сельский участок", 1.3),
]

CATALOG = [
    ("KPI-01", "Раскрытие тяжких преступлений", "Уголовный розыск", "УК РК, ст. 188", 12, "дело"),
    ("KPI-02", "Профилактика правонарушений", "Профилактика", "КоАП РК, ст. 434", 3, "мероприятие"),
    ("KPI-03", "Оформление ДТП", "Дорожная безопасность", "КоАП РК, ст. 590", 4, "протокол"),
    ("KPI-04", "Выявление адм. правонарушений", "Административная практика", "КоАП РК", 2, "протокол"),
    ("KPI-05", "Розыск лиц", "Розыск", "УПК РК, ст. 271", 8, "лицо"),
    ("KPI-06", "Изъятие оружия и наркотиков", "Оборот веществ", "УК РК, ст. 296", 14, "факт"),
    ("KPI-07", "Работа с обращениями граждан", "Взаимодействие с населением", "Закон 'Об ОВД'", 1, "обращение"),
    ("KPI-08", "Патрулирование территории", "Охрана порядка", "Закон 'Об ОВД'", 1, "смена"),
]

# (username, password, ФИО, звание, роль, индекс участка)
USERS = [
    ("head", "head123", "Нурланов Бекзат Сериккали", "Полковник полиции", "head", 0),
    ("aibek", "pass123", "Ахметов Айбек Маратұлы", "Лейтенант полиции", "officer", 1),
    ("dana", "pass123", "Сулейменова Дана Қанатқызы", "Капитан полиции", "officer", 2),
    ("ruslan", "pass123", "Жумабеков Руслан Ермекұлы", "Старший лейтенант полиции", "officer", 3),
    ("aliya", "pass123", "Оспанова Әлия Бауыржанқызы", "Майор полиции", "officer", 4),
]

# Примеры записей: (username, kpi_code, quantity, days_ago, status)
SAMPLE_ENTRIES = [
    ("aibek", "KPI-04", 6, 40, "confirmed"),
    ("aibek", "KPI-08", 12, 35, "confirmed"),
    ("aibek", "KPI-02", 3, 20, "confirmed"),
    ("aibek", "KPI-03", 4, 10, "pending"),
    ("dana", "KPI-01", 2, 45, "confirmed"),
    ("dana", "KPI-05", 3, 30, "confirmed"),
    ("dana", "KPI-06", 1, 15, "confirmed"),
    ("dana", "KPI-07", 9, 5, "pending"),
    ("ruslan", "KPI-08", 15, 38, "confirmed"),
    ("ruslan", "KPI-04", 4, 22, "confirmed"),
    ("ruslan", "KPI-03", 7, 12, "rejected"),
    ("ruslan", "KPI-02", 2, 3, "pending"),
    ("aliya", "KPI-01", 3, 50, "confirmed"),
    ("aliya", "KPI-06", 2, 28, "confirmed"),
    ("aliya", "KPI-05", 4, 14, "confirmed"),
    ("aliya", "KPI-07", 11, 4, "pending"),
]

# ---------- Генерация дополнительных сотрудников ----------
MALE_FIRST = [
    "Айбек", "Руслан", "Бекзат", "Ерлан", "Нурлан", "Дамир", "Арман", "Темирлан",
    "Алихан", "Дидар", "Санжар", "Ержан", "Мадияр", "Олжас", "Бауыржан", "Канат",
    "Серик", "Талгат", "Аскар", "Даурен", "Жанибек", "Нуржан", "Рамазан", "Сабыр",
    "Ерасыл", "Нурсултан", "Аян", "Дастан", "Мирас", "Алмат",
]
FEMALE_FIRST = [
    "Дана", "Әлия", "Айгерим", "Камила", "Динара", "Жанна", "Сауле", "Гульнара",
    "Айжан", "Мадина", "Асель", "Жулдыз", "Назгуль", "Ляззат", "Ботагоз", "Айнур",
    "Гаухар", "Зарина", "Меруерт", "Нургуль", "Аружан", "Томирис", "Алуа", "Жанель",
    "Карина", "Сабина", "Динара", "Балжан",
]
SURNAME_ROOTS = [
    "Ахметов", "Сулейменов", "Жумабеков", "Оспанов", "Нурланов", "Бекжанов",
    "Сериков", "Кенжебеков", "Алтынбеков", "Досжанов", "Мукашев", "Тулегенов",
    "Абдрахманов", "Искаков", "Байжанов", "Сатпаев", "Ермеков", "Калиев",
    "Турсынов", "Жаксыбеков", "Омаров", "Касымов", "Айтбаев", "Рахимов",
    "Сейтказин", "Молдабеков", "Утегенов", "Карибаев", "Смагулов", "Бекмуратов",
]
RANKS = [
    "Лейтенант полиции", "Старший лейтенант полиции", "Капитан полиции",
    "Майор полиции", "Подполковник полиции", "Сержант полиции",
    "Старший сержант полиции",
]


def _feminize_surname(s):
    for suf in ("ов", "ев", "ин"):
        if s.endswith(suf):
            return s + "а"
    return s + "а"


def _make_officer(idx, rng):
    """Возвращает (username, password, ФИО, звание, role, district_idx)."""
    male = rng.random() < 0.6
    surname = rng.choice(SURNAME_ROOTS)
    if not male:
        surname = _feminize_surname(surname)
    first = rng.choice(MALE_FIRST if male else FEMALE_FIRST)
    father = rng.choice(MALE_FIRST)
    patronymic = father + ("ұлы" if male else "қызы")
    fio = f"{surname} {first} {patronymic}"
    rank = rng.choice(RANKS)
    district_idx = 1 + (idx % (len(DISTRICTS) - 1))  # участки 1..N (0 — для начальника)
    return (f"officer{idx}", "pass123", fio, rank, "officer", district_idx)


def _generate(seed_value=42):
    """Доводит число сотрудников до TARGET_OFFICERS и создаёт ровно TARGET_KPI_ENTRIES записей KPI."""
    rng = random.Random(seed_value)
    named_officers = [u for u in USERS if u[4] == "officer"]
    named_count = len(named_officers)

    extra_users = []
    for i in range(named_count + 1, TARGET_OFFICERS + 1):
        extra_users.append(_make_officer(i, rng))

    officer_usernames = [u[0] for u in named_officers] + [u[0] for u in extra_users]
    codes = [c[0] for c in CATALOG]
    statuses = ["confirmed"] * 6 + ["pending"] * 3 + ["rejected"]

    remaining = TARGET_KPI_ENTRIES - len(SAMPLE_ENTRIES)
    assigned_usernames = officer_usernames.copy()
    while len(assigned_usernames) < remaining:
        assigned_usernames.extend(officer_usernames)
    assigned_usernames = assigned_usernames[:remaining]
    rng.shuffle(assigned_usernames)

    extra_entries = []
    for username in assigned_usernames:
        code = rng.choice(codes)
        qty = rng.randint(1, 18)
        days_ago = rng.randint(0, 364)  # полный год
        status = rng.choice(statuses)
        extra_entries.append((username, code, qty, days_ago, status))

    return extra_users, extra_entries


def seed():
    Base.metadata.create_all(bind=engine)
    db = SessionLocal()
    if db.query(User).count() > 0:
        db.close()
        return  # уже заполнено

    districts = [District(name=n, coeff=c) for n, c in DISTRICTS]
    db.add_all(districts)
    db.flush()

    catalog = [
        KpiCatalog(code=c, indicator=i, block=b, npa=n, points_per_unit=p, unit=u)
        for c, i, b, n, p, u in CATALOG
    ]
    db.add_all(catalog)
    db.flush()

    extra_users, extra_entries = _generate()
    all_users = USERS + extra_users
    all_entries = SAMPLE_ENTRIES + extra_entries

    users = {}
    for username, pwd, fio, rank, role, d_idx in all_users:
        u = User(
            username=username,
            password_hash=hash_password(pwd),
            full_name=fio,
            rank=rank,
            role=role,
            district_id=districts[d_idx].id,
        )
        db.add(u)
        users[username] = u
    db.flush()

    catalog_by_code = {c.code: c for c in catalog}
    today = date.today()
    for username, code, qty, days_ago, status in all_entries:
        u = users[username]
        cat = catalog_by_code[code]
        coeff = u.district.coeff if u.district else 1.0
        raw = cat.points_per_unit * qty
        d = today - timedelta(days=days_ago)
        db.add(KpiEntry(
            date=d,
            month=d.strftime("%Y-%m"),
            officer_id=u.id,
            catalog_id=cat.id,
            quantity=qty,
            points_per_unit=cat.points_per_unit,
            coeff=coeff,
            raw_score=raw,
            final_score=round(raw * coeff, 2),
            status=status,
            comments="",
            head_comment="Подтверждено" if status == "confirmed" else ("Недостаточно данных" if status == "rejected" else ""),
        ))

    db.commit()
    db.close()
    print("База данных заполнена начальными данными.")


if __name__ == "__main__":
    seed()
