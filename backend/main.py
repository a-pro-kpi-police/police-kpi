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
from models import District, KpiCatalog, KpiEntry, User
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
    # Сотрудник видит только свои записи; начальник — все.
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
    cat = db.query(KpiCatalog).get(payload.catalog_id)
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
    entry = db.query(KpiEntry).get(entry_id)
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

    # Рейтинг по сотрудникам (по подтверждённым итоговым баллам)
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

    # Разбивка по блокам (подтверждённые)
    by_block = defaultdict(float)
    for e in confirmed:
        by_block[e.catalog.block if e.catalog else "—"] += e.final_score
    blocks = [{"block": b, "final": round(v, 2)} for b, v in sorted(by_block.items(), key=lambda x: -x[1])]

    # Динамика по месяцам (подтверждённые)
    by_month = defaultdict(float)
    for e in confirmed:
        by_month[e.month] += e.final_score
    trend = [{"month": m, "final": round(v, 2)} for m, v in sorted(by_month.items())]

    # Поток решений по месяцам помогает увидеть накопление неподтверждённых записей.
    by_month_status = defaultdict(lambda: {"confirmed": 0, "pending": 0, "rejected": 0})
    for e in entries:
        by_month_status[e.month][e.status] += 1
    status_trend = [
        {"month": m, **counts}
        for m, counts in sorted(by_month_status.items())
    ]

    # Объём выполненной работы по конкретным KPI (не баллы, а физические единицы).
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

    # Очередь и отклонения по участкам выделяют зоны, требующие внимания руководителя.
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
