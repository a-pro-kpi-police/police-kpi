"""Начальное наполнение базы: участки, каталог KPI, пользователи и примеры записей."""
import random
from datetime import date, timedelta

from database import SessionLocal, engine, Base
from models import District, User, KpiCatalog, KpiEntry, KpiTarget
from auth import hash_password

TARGET_OFFICERS = 20
TARGET_KPI_ENTRIES = 1000

DISTRICTS = [
    ("Обычный участок", 1.0),
    ("Повышенной сложности", 1.1),
    ("Сложный участок", 1.2),
    ("Особо сложный участок", 1.3),
]

CATALOG = [
    ("KPI-01", "КоАП РК ст.127 ч.1", "Административная практика", "КоАП РК ст.127 ч.1", 6, "факт"),
    ("KPI-02", "КоАП РК ст.127 ч.2", "Административная практика", "КоАП РК ст.127 ч.2", 7, "факт"),
    ("KPI-03", "КоАП РК ст.127 ч.3", "Административная практика", "КоАП РК ст.127 ч.3", 7, "факт"),
    ("KPI-04", "КоАП РК ст.127-1", "Административная практика", "КоАП РК ст.127-1", 7, "факт"),
    ("KPI-05", "КоАП РК ст.127-2", "Административная практика", "КоАП РК ст.127-2", 7, "факт"),
    ("KPI-06", "КоАП РК ст.128", "Административная практика", "КоАП РК ст.128", 6, "факт"),
    ("KPI-07", "КоАП РК ст.131", "Административная практика", "КоАП РК ст.131", 5, "факт"),
    ("KPI-08", "КоАП РК ст.132", "Административная практика", "КоАП РК ст.132", 4, "факт"),
    ("KPI-09", "КоАП РК ст.133", "Административная практика", "КоАП РК ст.133", 4, "факт"),
    ("KPI-10", "КоАП РК ст.200 ч.1", "Административная практика", "КоАП РК ст.200 ч.1", 4, "факт"),
    ("KPI-11", "КоАП РК ст.200 ч.2", "Административная практика", "КоАП РК ст.200 ч.2", 5, "факт"),
    ("KPI-12", "КоАП РК ст.441-1", "Административная практика", "КоАП РК ст.441-1", 2, "факт"),
    ("KPI-13", "КоАП РК ст.442 ч.1", "Административная практика", "КоАП РК ст.442 ч.1", 4, "факт"),
    ("KPI-14", "КоАП РК ст.442 ч.2", "Административная практика", "КоАП РК ст.442 ч.2", 5, "факт"),
    ("KPI-15", "КоАП РК ст.442 ч.3", "Административная практика", "КоАП РК ст.442 ч.3", 6, "факт"),
    ("KPI-16", "КоАП РК ст.434", "Административная практика", "КоАП РК ст.434", 5, "факт"),
    ("KPI-17", "КоАП РК ст.435", "Административная практика", "КоАП РК ст.435", 6, "факт"),
    ("KPI-18", "КоАП РК ст.440", "Административная практика", "КоАП РК ст.440", 7, "факт"),
    ("KPI-19", "КоАП РК ст.444", "Административная практика", "КоАП РК ст.444", 6, "факт"),
    ("KPI-20", "КоАП РК ст.448", "Административная практика", "КоАП РК ст.448", 7, "факт"),
    ("KPI-21", "Профилактическая беседа с несовершеннолетним", "Профилактика несовершеннолетних", "", 2, "беседа"),
    ("KPI-22", "Профилактическая беседа с родителем (законным представителем)", "Профилактика несовершеннолетних", "", 3, "беседа"),
    ("KPI-23", "Лекция/выступление в школе, колледже", "Профилактика несовершеннолетних", "", 3, "факт"),
    ("KPI-24", "Родительское собрание", "Профилактика несовершеннолетних", "", 4, "факт"),
    ("KPI-25", "Рейд/ОПМ без выявлений", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-26", "Рейд/ОПМ с выявлением и принятием мер", "Профилактика несовершеннолетних", "", 8, "факт"),
    ("KPI-27", "Представление об устранении причин и условий", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-28", "Совместная работа со школой, психологом, органом опеки, акиматом", "Профилактика несовершеннолетних", "", 8, "факт"),
    ("KPI-29", "Выявление троллинга/кибербуллинга", "Профилактика несовершеннолетних", "", 8, "факт"),
    ("KPI-30", "Выявление кризисного риска для жизни и здоровья ребенка", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-31", "Организация занятости несовершеннолетнего", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-32", "Месячный профилактический план выполнен без замечаний", "Профилактика несовершеннолетних", "", 4, "факт"),
    ("KPI-33", "Постановка несовершеннолетнего на профилактический учет", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-34", "Постановка родителя на профилактический контроль", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-35", "Проверка подучетного/семьи по месту жительства", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-36", "Индивидуальный план профилактики оформлен без замечаний", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-37", "Корректировка индивидуального плана", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-38", "Снятие с учета по положительным основаниям", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-39", "Повторное правонарушение подучетного", "Профилактика несовершеннолетних", "", -2, "факт"),
    ("KPI-40", "Отсутствие повторных правонарушений среди подучетных", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-41", "Выявление неблагополучной семьи", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-42", "Положительная динамика по семье/несовершеннолетнему", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-43", "Выезд по сообщению с участием несовершеннолетнего", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-44", "Ночной/неотложный выезд", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-45", "Семейно-бытовой конфликт", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-46", "Доставление несовершеннолетнего в ЦАН/центр защиты", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-47", "Материал направлен в КДН, комиссия удовлетворила материал", "Профилактика несовершеннолетних", "", 10, "факт"),
    ("KPI-48", "Мероприятия по самовольному уходу/розыску", "Профилактика несовершеннолетних", "", 8, "факт"),
    ("KPI-49", "Установление местонахождения и возвращение несовершеннолетнего", "Профилактика несовершеннолетних", "", 8, "факт"),
    ("KPI-50", "Материал оформлен без замечаний руководителя/суда/КДН", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-51", "Возврат материала на доработку по вине сотрудника", "Профилактика несовершеннолетних", "", -2, "факт"),
    ("KPI-52", "Просрочка материала/ответа", "Профилактика несовершеннолетних", "", -3, "факт"),
    ("KPI-53", "Отмена материала из-за ошибок оформления", "Профилактика несовершеннолетних", "", -4, "факт"),
    ("KPI-54", "Своевременная отработка Е-өтініш/КУИ", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-55", "Проверка учетного дела без замечаний", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-56", "Своевременное внесение данных в ИПС/ЕРАП", "Профилактика несовершеннолетних", "", 4, "факт"),
    ("KPI-57", "Ошибка в ИПС/ЕРАП/учете", "Профилактика несовершеннолетних", "", -2, "факт"),
    ("KPI-58", "Неактуальное ведение учета", "Профилактика несовершеннолетних", "", -2, "факт"),
    ("KPI-59", "Полный комплект подтверждающих документов", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-60", "Запросы и взаимодействие со школой, психологом, органом опеки", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-61", "Участие в заседании КДН/межведомственной комиссии", "Профилактика несовершеннолетних", "", 7, "факт"),
    ("KPI-62", "Благодарность/положительный отзыв", "Профилактика несовершеннолетних", "", 6, "факт"),
    ("KPI-63", "Подтвержденная жалоба на сотрудника", "Профилактика несовершеннолетних", "", -5, "факт"),
    ("KPI-64", "Подтвержденная негативная публикация по вине сотрудника", "Профилактика несовершеннолетних", "", -5, "факт"),
    ("KPI-65", "Опоздание на службу", "Профилактика несовершеннолетних", "", -1, "факт"),
    ("KPI-66", "Нарушение формы одежды", "Профилактика несовершеннолетних", "", -1, "факт"),
    ("KPI-67", "Отсутствие жетона/удостоверения", "Профилактика несовершеннолетних", "", -1, "факт"),
    ("KPI-68", "Дисциплинарное взыскание", "Профилактика несовершеннолетних", "", -5, "факт"),
    ("KPI-69", "Отсутствие взысканий и замечаний за месяц", "Профилактика несовершеннолетних", "", 5, "факт"),
    ("KPI-70", "Неисполнение поручения/отчета в срок", "Профилактика несовершеннолетних", "", -3, "факт"),
]

USERS = [
    ("head", "head123", "Нурланов Бекзат Сериккали", "Полковник полиции", "head", 0),
    ("aibek", "pass123", "Ахметов Айбек Маратұлы", "Лейтенант полиции", "officer", 0),
    ("dana", "pass123", "Сулейменова Дана Қанатқызы", "Капитан полиции", "officer", 1),
    ("ruslan", "pass123", "Жумабеков Руслан Ермекұлы", "Старший лейтенант полиции", "officer", 2),
    ("aliya", "pass123", "Оспанова Әлия Бауыржанқызы", "Майор полиции", "officer", 3),
]

SAMPLE_ENTRIES = [
    ("aibek", "KPI-04", 1, 40, "confirmed"),
    ("aibek", "KPI-08", 1, 35, "confirmed"),
    ("aibek", "KPI-02", 1, 20, "confirmed"),
    ("aibek", "KPI-03", 1, 10, "pending"),
    ("dana", "KPI-01", 1, 45, "confirmed"),
    ("dana", "KPI-05", 1, 30, "confirmed"),
    ("dana", "KPI-06", 1, 15, "confirmed"),
    ("dana", "KPI-07", 1, 5, "pending"),
    ("ruslan", "KPI-08", 1, 38, "confirmed"),
    ("ruslan", "KPI-04", 1, 22, "confirmed"),
    ("ruslan", "KPI-03", 1, 12, "rejected"),
    ("ruslan", "KPI-02", 1, 3, "pending"),
    ("aliya", "KPI-01", 1, 50, "confirmed"),
    ("aliya", "KPI-06", 1, 28, "confirmed"),
    ("aliya", "KPI-05", 1, 14, "confirmed"),
    ("aliya", "KPI-07", 1, 4, "pending"),
]

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
    male = rng.random() < 0.6
    surname = rng.choice(SURNAME_ROOTS)
    if not male:
        surname = _feminize_surname(surname)
    first = rng.choice(MALE_FIRST if male else FEMALE_FIRST)
    father = rng.choice(MALE_FIRST)
    patronymic = father + ("ұлы" if male else "қызы")
    fio = f"{surname} {first} {patronymic}"
    rank = rng.choice(RANKS)
    district_idx = idx % len(DISTRICTS)
    return (f"officer{idx}", "pass123", fio, rank, "officer", district_idx)


def _generate(seed_value=42):
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
        qty = 1
        days_ago = rng.randint(0, 364)
        status = rng.choice(statuses)
        extra_entries.append((username, code, qty, days_ago, status))

    return extra_users, extra_entries


def _month_shift(base: date, offset: int) -> str:
    total_months = base.year * 12 + (base.month - 1) + offset
    year = total_months // 12
    month = total_months % 12 + 1
    return f"{year:04d}-{month:02d}"


def _default_target_quantity(confirmed_quantity: float, officer_index: int, catalog_index: int, month_index: int) -> float:
    profile = officer_index % 3
    if confirmed_quantity > 0:
        if profile == 0:
            return round(confirmed_quantity, 2)
        if profile == 1:
            return round(max(confirmed_quantity - 0.5, 0.5), 2)
        extra = 1.0 if (catalog_index + month_index) % 2 == 0 else 0.5
        return round(confirmed_quantity + extra, 2)
    if profile == 2 and (catalog_index + month_index) % 4 == 0:
        return 1.0
    return 0.0


def _ensure_targets(db):
    officers = db.query(User).filter(User.role == "officer").order_by(User.id).all()
    catalog = db.query(KpiCatalog).filter(
        KpiCatalog.points_per_unit >= 0,
    ).order_by(KpiCatalog.code).all()
    if not officers or not catalog:
        return

    months = [_month_shift(date.today(), 0), _month_shift(date.today(), -1)]
    existing = {
        (target.month, target.officer_id, target.catalog_id)
        for target in db.query(KpiTarget).filter(KpiTarget.month.in_(months)).all()
    }
    confirmed_quantities = {}
    confirmed_entries = db.query(KpiEntry).filter(
        KpiEntry.status == "confirmed",
        KpiEntry.month.in_(months),
    ).all()
    for entry in confirmed_entries:
        key = (entry.month, entry.officer_id, entry.catalog_id)
        confirmed_quantities[key] = round(confirmed_quantities.get(key, 0.0) + entry.quantity, 2)

    changed = False
    for officer_index, officer in enumerate(officers):
        for month_index, month in enumerate(months):
            for catalog_index, item in enumerate(catalog):
                key = (month, officer.id, item.id)
                if key in existing:
                    continue
                db.add(KpiTarget(
                    month=month,
                    officer_id=officer.id,
                    catalog_id=item.id,
                    target_quantity=_default_target_quantity(
                        confirmed_quantities.get(key, 0.0),
                        officer_index,
                        catalog_index,
                        month_index,
                    ),
                ))
                changed = True
    if changed:
        db.commit()


def _sync_catalog(db):
    """Обновляет справочник без удаления связанных исторических записей."""
    existing = {item.code: item for item in db.query(KpiCatalog).all()}
    for code, indicator, block, npa, points, unit in CATALOG:
        item = existing.get(code)
        if item:
            item.indicator = indicator
            item.block = block
            item.npa = npa
            item.points_per_unit = points
            item.unit = unit
        else:
            db.add(KpiCatalog(
                code=code,
                indicator=indicator,
                block=block,
                npa=npa,
                points_per_unit=points,
                unit=unit,
            ))
    db.commit()


def seed():
    Base.metadata.create_all(bind=engine)
    db = SessionLocal()
    if db.query(User).count() > 0:
        districts = db.query(District).order_by(District.id).all()
        if len(districts) == 5 and districts[0].name == "Центральный участок":
            central, districts = districts[0], districts[1:]
            db.query(User).filter(User.district_id == central.id).update(
                {User.district_id: districts[0].id}, synchronize_session=False
            )
            db.delete(central)
        if len(districts) == len(DISTRICTS):
            for district, (name, coeff) in zip(districts, DISTRICTS):
                district.name = name
                district.coeff = coeff
            db.commit()
        _sync_catalog(db)
        _ensure_targets(db)
        db.close()
        return

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
    _ensure_targets(db)
    db.close()
    print("База данных заполнена начальными данными.")


if __name__ == "__main__":
    seed()
