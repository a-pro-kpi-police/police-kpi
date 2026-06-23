"""KPI-дашборд отдела полиции — FastAPI backend."""
from collections import defaultdict
from datetime import date
from typing import List, Optional

from fastapi import Depends, FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.security import OAuth2PasswordRequestForm
from sqlalchemy.orm import Session

import schemas
from auth import (create_token, get_current_user, require_head, verify_password)
from database import get_db
from models import District, KpiCatalog, KpiEntry, KpiTarget, User
from seed import seed

app = FastAPI(title="KPI отдела полиции", version="1.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
def on_startup():
    seed()


def entry_to_out(e: KpiEntry) -> schemas.EntryOut:
    return schemas.EntryOut(
        id=e.id,
        date=e.date,
        month=e.month,
        officer_id=e.officer_id,
        officer_name=e.officer.full_name if e.officer else "",
        rank=e.officer.rank if e.officer else "",
        district_name=e.officer.district.name if e.officer and e.officer.district else "",
        code=e.catalog.code if e.catalog else "",
        indicator=e.catalog.indicator if e.catalog else "",
        block=e.catalog.block if e.catalog else "",
        npa=e.catalog.npa if e.catalog else "",
        quantity=e.quantity,
        points_per_unit=e.points_per_unit,
        coeff=e.coeff,
        raw_score=e.raw_score,
        final_score=e.final_score,
        status=e.status,
        comments=e.comments or "",
        head_comment=e.head_comment or "",
    )


def validate_month(month: str):
    if not isinstance(month, str) or len(month) != 7 or month[4] != "-":
        raise HTTPException(status_code=400, detail="Месяц указан неверно")
    year, month_num = month.split("-", 1)
    if not year.isdigit() or not month_num.isdigit() or not (1 <= int(month_num) <= 12):
        raise HTTPException(status_code=400, detail="Месяц указан неверно")


def get_officer_or_404(db: Session, officer_id: int) -> User:
    officer = db.get(User, officer_id)
    if not officer or officer.role != "officer":
        raise HTTPException(status_code=404, detail="Сотрудник не найден")
    return officer


def build_target_rows(db: Session, officer: User, month: str) -> list[schemas.TargetOut]:
    validate_month(month)
    coeff = officer.district.coeff if officer.district else 1.0
    targets = db.query(KpiTarget).filter(
        KpiTarget.officer_id == officer.id,
        KpiTarget.month == month,
    ).all()
    quantities = {target.catalog_id: target.target_quantity for target in targets}

    rows = []
    for catalog in db.query(KpiCatalog).order_by(KpiCatalog.code).all():
        target_quantity = round(quantities.get(catalog.id, 0), 2)
        rows.append(schemas.TargetOut(
            catalog_id=catalog.id,
            code=catalog.code,
            indicator=catalog.indicator,
            unit=catalog.unit,
            points_per_unit=catalog.points_per_unit,
            target_quantity=target_quantity,
            target_score=round(target_quantity * catalog.points_per_unit * coeff, 2),
        ))
    return rows


def build_target_progress(db: Session, officer: User, month: str) -> schemas.TargetProgressOut:
    rows = build_target_rows(db, officer, month)
    confirmed_entries = db.query(KpiEntry).filter(
        KpiEntry.officer_id == officer.id,
        KpiEntry.month == month,
        KpiEntry.status == "confirmed",
    ).all()

    actual_by_catalog = defaultdict(lambda: {"quantity": 0.0, "score": 0.0})
    for entry in confirmed_entries:
        actual_by_catalog[entry.catalog_id]["quantity"] += entry.quantity
        actual_by_catalog[entry.catalog_id]["score"] += entry.final_score

    items = []
    actual_total = 0.0
    target_total = 0.0
    for row in rows:
        actual = actual_by_catalog[row.catalog_id]
        confirmed_quantity = round(actual["quantity"], 2)
        confirmed_score = round(actual["score"], 2)
        remaining_quantity = round(max(row.target_quantity - confirmed_quantity, 0), 2)
        achievement_percent = round(100 * confirmed_score / row.target_score, 1) if row.target_score else 0.0
        items.append(schemas.TargetProgressItemOut(
            catalog_id=row.catalog_id,
            code=row.code,
            indicator=row.indicator,
            unit=row.unit,
            points_per_unit=row.points_per_unit,
            target_quantity=row.target_quantity,
            confirmed_quantity=confirmed_quantity,
            remaining_quantity=remaining_quantity,
            target_score=row.target_score,
            confirmed_score=confirmed_score,
            achievement_percent=achievement_percent,
        ))
        actual_total += confirmed_score
        target_total += row.target_score

    actual_total = round(actual_total, 2)
    target_total = round(target_total, 2)
    return schemas.TargetProgressOut(
        month=month,
        officer_id=officer.id,
        actual_score=actual_total,
        target_score=target_total,
        achievement_percent=round(100 * actual_total / target_total, 1) if target_total else 0.0,
        items=items,
    )


def build_target_overview(db: Session, month: str, officer_id: Optional[int] = None) -> schemas.TargetOverviewOut:
    validate_month(month)
    officers = [get_officer_or_404(db, officer_id)] if officer_id else (
        db.query(User).filter(User.role == "officer").order_by(User.full_name).all()
    )

    rows = []
    items = []
    actual_total = 0.0
    target_total = 0.0
    on_target_count = 0

    for officer in officers:
        progress = build_target_progress(db, officer, month)
        gap = round(progress.actual_score - progress.target_score, 2)
        status = "no_target" if progress.target_score == 0 else (
            "on_target" if progress.actual_score >= progress.target_score else "behind"
        )
        if status == "on_target":
            on_target_count += 1
        rows.append(schemas.TargetOverviewOfficerOut(
            officer_id=officer.id,
            name=officer.full_name,
            actual_score=progress.actual_score,
            target_score=progress.target_score,
            achievement_percent=progress.achievement_percent,
            gap=gap,
            status=status,
        ))
        actual_total += progress.actual_score
        target_total += progress.target_score
        if officer_id:
            items = progress.items

    if not officer_id:
        status_order = {"behind": 0, "no_target": 1, "on_target": 2}
        rows.sort(key=lambda row: (status_order.get(row.status, 9), row.achievement_percent if row.target_score else 9999, row.gap, row.name))

    actual_total = round(actual_total, 2)
    target_total = round(target_total, 2)
    return schemas.TargetOverviewOut(
        month=month,
        actual_score=actual_total,
        target_score=target_total,
        achievement_percent=round(100 * actual_total / target_total, 1) if target_total else 0.0,
        on_target_count=on_target_count,
        officers_count=len(rows),
        officers=rows,
        items=items,
    )


# ---------- Аутентификация ----------
@app.post("/api/auth/login", response_model=schemas.Token)
def login(form: OAuth2PasswordRequestForm = Depends(), db: Session = Depends(get_db)):
    user = db.query(User).filter(User.username == form.username).first()
    if not user or not verify_password(form.password, user.password_hash):
        raise HTTPException(status_code=401, detail="Неверный логин или пароль")
    return schemas.Token(access_token=create_token(user))


@app.get("/api/auth/me", response_model=schemas.UserOut)
def me(user: User = Depends(get_current_user)):
    return user


# ---------- Справочники ----------
@app.get("/api/catalog", response_model=List[schemas.CatalogOut])
def get_catalog(db: Session = Depends(get_db), user: User = Depends(get_current_user)):
    return db.query(KpiCatalog).order_by(KpiCatalog.code).all()


@app.get("/api/districts", response_model=List[schemas.DistrictOut])
def get_districts(db: Session = Depends(get_db), user: User = Depends(get_current_user)):
    return db.query(District).all()


@app.get("/api/officers", response_model=List[schemas.UserOut])
def list_officers(db: Session = Depends(get_db), head: User = Depends(require_head)):
    return db.query(User).filter(User.role == "officer").order_by(User.full_name).all()


# ---------- Целевые KPI ----------
@app.get("/api/targets", response_model=List[schemas.TargetOut])
def get_targets(
    month: str = Query(...),
    officer_id: int = Query(...),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    officer = get_officer_or_404(db, officer_id)
    return build_target_rows(db, officer, month)


@app.put("/api/targets", response_model=List[schemas.TargetOut])
def save_targets(
    payload: schemas.TargetSavePayload,
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    officer = get_officer_or_404(db, payload.officer_id)
    validate_month(payload.month)

    catalog_ids = {catalog_id for (catalog_id,) in db.query(KpiCatalog.id).all()}
    new_quantities = {}
    for item in payload.items:
        if item.catalog_id not in catalog_ids:
            raise HTTPException(status_code=404, detail="Показатель KPI не найден")
        if item.target_quantity > 0:
            new_quantities[item.catalog_id] = item.target_quantity

    existing = db.query(KpiTarget).filter(
        KpiTarget.officer_id == officer.id,
        KpiTarget.month == payload.month,
    ).all()
    existing_by_catalog = {target.catalog_id: target for target in existing}

    for target in existing:
        if target.catalog_id not in new_quantities:
            db.delete(target)

    for catalog_id, target_quantity in new_quantities.items():
        current = existing_by_catalog.get(catalog_id)
        if current:
            current.target_quantity = target_quantity
        else:
            db.add(KpiTarget(
                month=payload.month,
                officer_id=officer.id,
                catalog_id=catalog_id,
                target_quantity=target_quantity,
            ))

    db.commit()
    return build_target_rows(db, officer, payload.month)


@app.get("/api/target-progress", response_model=schemas.TargetProgressOut)
def target_progress(
    month: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    selected_month = month or date.today().strftime("%Y-%m")
    validate_month(selected_month)
    if user.role == "head":
        if not officer_id:
            raise HTTPException(status_code=400, detail="Сотрудник не указан")
        officer = get_officer_or_404(db, officer_id)
    else:
        officer = user
    if user.role != "head" and officer.id != user.id:
        raise HTTPException(status_code=403, detail="Недостаточно прав")
    return build_target_progress(db, officer, selected_month)


@app.get("/api/target-overview", response_model=schemas.TargetOverviewOut)
def target_overview(
    month: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    selected_month = month or date.today().strftime("%Y-%m")
    return build_target_overview(db, selected_month, officer_id)


# ---------- Записи KPI ----------
@app.get("/api/entries", response_model=List[schemas.EntryOut])
def list_entries(
    month: Optional[str] = Query(None),
    status: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    q = db.query(KpiEntry)
    if user.role != "head":
        q = q.filter(KpiEntry.officer_id == user.id)
    elif officer_id:
        q = q.filter(KpiEntry.officer_id == officer_id)
    if month:
        q = q.filter(KpiEntry.month == month)
    if status:
        q = q.filter(KpiEntry.status == status)
    entries = q.order_by(KpiEntry.date.desc(), KpiEntry.id.desc()).all()
    return [entry_to_out(e) for e in entries]


@app.post("/api/entries", response_model=schemas.EntryOut)
def create_entry(
    payload: schemas.EntryCreate,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    if user.role != "officer":
        raise HTTPException(status_code=403, detail="Недостаточно прав")
    cat = db.get(KpiCatalog, payload.catalog_id)
    if not cat:
        raise HTTPException(status_code=404, detail="Показатель KPI не найден")
    if payload.quantity <= 0:
        raise HTTPException(status_code=400, detail="Количество должно быть больше нуля")

    coeff = user.district.coeff if user.district else 1.0
    raw = cat.points_per_unit * payload.quantity
    d = payload.date or date.today()

    entry = KpiEntry(
        date=d,
        month=d.strftime("%Y-%m"),
        officer_id=user.id,
        catalog_id=cat.id,
        quantity=payload.quantity,
        points_per_unit=cat.points_per_unit,
        coeff=coeff,
        raw_score=raw,
        final_score=round(raw * coeff, 2),
        status="pending",
        comments=payload.comments,
        head_comment="",
    )
    db.add(entry)
    db.commit()
    db.refresh(entry)
    return entry_to_out(entry)


@app.patch("/api/entries/{entry_id}/status", response_model=schemas.EntryOut)
def update_status(
    entry_id: int,
    payload: schemas.EntryStatusUpdate,
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    entry = db.get(KpiEntry, entry_id)
    if not entry:
        raise HTTPException(status_code=404, detail="Запись не найдена")
    if payload.status not in ("pending", "confirmed", "rejected"):
        raise HTTPException(status_code=400, detail="Недопустимый статус")
    entry.status = payload.status
    entry.head_comment = payload.head_comment
    db.commit()
    db.refresh(entry)
    return entry_to_out(entry)


# ---------- Дашборд ----------
@app.get("/api/dashboard")
def dashboard(
    month: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    q = db.query(KpiEntry)
    if officer_id:
        q = q.filter(KpiEntry.officer_id == officer_id)
    if month:
        q = q.filter(KpiEntry.month == month)
    entries = q.all()

    confirmed = [e for e in entries if e.status == "confirmed"]
    pending = [e for e in entries if e.status == "pending"]
    rejected = [e for e in entries if e.status == "rejected"]

    total_final = round(sum(e.final_score for e in confirmed), 2)

    by_officer = defaultdict(lambda: {"final": 0.0, "raw": 0.0, "confirmed": 0, "pending": 0, "name": "", "rank": "", "district": "", "blocks": defaultdict(float)})
    for e in entries:
        o = by_officer[e.officer_id]
        o["name"] = e.officer.full_name if e.officer else ""
        o["rank"] = e.officer.rank if e.officer else ""
        o["district"] = e.officer.district.name if e.officer and e.officer.district else ""
        if e.status == "confirmed":
            o["final"] += e.final_score
            o["raw"] += e.raw_score
            o["confirmed"] += 1
            block = e.catalog.block if e.catalog else "—"
            o["blocks"][block] += e.final_score
        elif e.status == "pending":
            o["pending"] += 1
    officers = [
        {
            "officer_id": oid,
            "name": d["name"], "rank": d["rank"], "district": d["district"],
            "final": round(d["final"], 2), "raw": round(d["raw"], 2),
            "confirmed": d["confirmed"], "pending": d["pending"],
            "blocks": {b: round(v, 2) for b, v in d["blocks"].items()},
        }
        for oid, d in by_officer.items()
    ]
    officers.sort(key=lambda x: x["final"], reverse=True)

    by_block = defaultdict(float)
    for e in confirmed:
        by_block[e.catalog.block if e.catalog else "—"] += e.final_score
    blocks = [{"block": b, "final": round(v, 2)} for b, v in sorted(by_block.items(), key=lambda x: -x[1])]

    by_month = defaultdict(float)
    for e in confirmed:
        by_month[e.month] += e.final_score
    trend = [{"month": m, "final": round(v, 2)} for m, v in sorted(by_month.items())]

    by_month_status = defaultdict(lambda: {"confirmed": 0, "pending": 0, "rejected": 0})
    for e in entries:
        by_month_status[e.month][e.status] += 1
    status_trend = [
        {"month": m, **counts}
        for m, counts in sorted(by_month_status.items())
    ]

    by_indicator = defaultdict(lambda: {"quantity": 0.0, "code": "", "indicator": ""})
    for e in entries:
        code = e.catalog.code if e.catalog else "—"
        item = by_indicator[code]
        item["code"] = code
        item["indicator"] = e.catalog.indicator if e.catalog else "—"
        item["quantity"] += e.quantity
    indicator_volume = [
        {**item, "quantity": round(item["quantity"], 2)}
        for item in sorted(by_indicator.values(), key=lambda value: -value["quantity"])
    ]

    by_district_review = defaultdict(lambda: {"pending": 0, "rejected": 0})
    for e in pending + rejected:
        district = e.officer.district.name if e.officer and e.officer.district else "—"
        by_district_review[district][e.status] += 1
    district_review = [
        {"district": name, **counts}
        for name, counts in sorted(
            by_district_review.items(),
            key=lambda item: -(item[1]["pending"] + item[1]["rejected"]),
        )
    ]

    total_decided = len(confirmed) + len(rejected)
    confirm_rate = round(100 * len(confirmed) / total_decided, 1) if total_decided else 0.0

    return {
        "total_final": total_final,
        "confirmed_count": len(confirmed),
        "pending_count": len(pending),
        "rejected_count": len(rejected),
        "officers_count": len(officers),
        "confirm_rate": confirm_rate,
        "officers": officers,
        "blocks": blocks,
        "trend": trend,
        "status_trend": status_trend,
        "indicator_volume": indicator_volume,
        "district_review": district_review,
    }


@app.get("/api/months")
def months(db: Session = Depends(get_db), user: User = Depends(get_current_user)):
    q = db.query(KpiEntry.month).distinct()
    if user.role != "head":
        q = q.filter(KpiEntry.officer_id == user.id)
    return sorted({m[0] for m in q.all()}, reverse=True)
