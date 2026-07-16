"""Маршруты записей KPI и их рассмотрения."""

from datetime import date
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user, require_head
from database import get_db
from models import KpiCatalog, KpiEntry, User
from services.entries import entry_to_out
from services.targets import get_officer_or_404

router = APIRouter(prefix="/api", tags=["entries"])


@router.get("/entries", response_model=list[schemas.EntryOut])
def list_entries(
    month: Optional[str] = Query(None),
    status: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    query = db.query(KpiEntry)
    if user.role != "head":
        query = query.filter(KpiEntry.officer_id == user.id)
    elif officer_id:
        query = query.filter(KpiEntry.officer_id == officer_id)
    if month:
        query = query.filter(KpiEntry.month == month)
    if status:
        query = query.filter(KpiEntry.status == status)
    entries = query.order_by(KpiEntry.date.desc(), KpiEntry.id.desc()).all()
    return [entry_to_out(entry) for entry in entries]


@router.post("/entries", response_model=schemas.EntryOut)
def create_entry(
    payload: schemas.EntryCreate,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    if user.role == "head":
        if not payload.officer_id:
            raise HTTPException(status_code=400, detail="Сотрудник не указан")
        officer = get_officer_or_404(db, payload.officer_id)
    elif user.role == "officer":
        if payload.officer_id is not None and payload.officer_id != user.id:
            raise HTTPException(status_code=403, detail="Недостаточно прав")
        officer = user
    else:
        raise HTTPException(status_code=403, detail="Недостаточно прав")

    catalog = db.get(KpiCatalog, payload.catalog_id)
    if not catalog:
        raise HTTPException(status_code=404, detail="Показатель KPI не найден")
    if user.role != "head" and catalog.points_per_unit < 0:
        raise HTTPException(
            status_code=403,
            detail="KPI с отрицательным баллом доступен только начальнику отдела",
        )
    if payload.quantity <= 0:
        raise HTTPException(
            status_code=400,
            detail="Количество должно быть больше нуля",
        )

    coeff = officer.district.coeff if officer.district else 1.0
    raw_score = catalog.points_per_unit * payload.quantity
    entry_date = payload.date or date.today()
    entry = KpiEntry(
        date=entry_date,
        month=entry_date.strftime("%Y-%m"),
        officer_id=officer.id,
        catalog_id=catalog.id,
        quantity=payload.quantity,
        points_per_unit=catalog.points_per_unit,
        coeff=coeff,
        raw_score=raw_score,
        final_score=round(raw_score * coeff, 2),
        status="pending",
        comments=payload.comments,
        head_comment="",
    )
    db.add(entry)
    db.commit()
    db.refresh(entry)
    return entry_to_out(entry)


@router.patch("/entries/{entry_id}/status", response_model=schemas.EntryOut)
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
