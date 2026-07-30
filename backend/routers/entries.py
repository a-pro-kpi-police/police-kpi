"""Маршруты записей KPI и их рассмотрения."""

from datetime import date
import os
from pathlib import Path
from typing import Optional
from uuid import uuid4

from fastapi import APIRouter, Depends, File, Form, HTTPException, Query, UploadFile
from fastapi.responses import FileResponse
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user, require_head
from database import get_db
from models import KpiCatalog, KpiEntry, User
from services.activity import add_notification, notify_heads
from services.entries import entry_to_out
from services.targets import get_officer_or_404

router = APIRouter(prefix="/api", tags=["entries"])

MAX_ATTACHMENT_SIZE = 20 * 1024 * 1024
ALLOWED_ATTACHMENT_EXTENSIONS = {
    ".csv", ".doc", ".docx", ".jpeg", ".jpg", ".pdf", ".png",
    ".txt", ".webp", ".xls", ".xlsx", ".zip",
}
DEFAULT_UPLOAD_DIR = Path(__file__).resolve().parent.parent / "uploads"
UPLOAD_DIR = Path(os.environ.get("KPI_UPLOAD_DIR", DEFAULT_UPLOAD_DIR))


def _build_entry(
    payload: schemas.EntryCreate,
    db: Session,
    user: User,
) -> KpiEntry:
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
    if not catalog or not catalog.is_active:
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
    return KpiEntry(
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


def _notify_entry_submitted(
    db: Session,
    user: User,
    entry: KpiEntry,
) -> None:
    if user.role != "officer":
        return
    catalog = db.get(KpiCatalog, entry.catalog_id)
    code = catalog.code if catalog else "KPI"
    notify_heads(
        db,
        kind="entry_pending",
        title="Новая запись KPI",
        message=f"{user.full_name} отправил запись {code} на подтверждение.",
        link="/entries",
    )


async def _store_attachment(upload: UploadFile) -> tuple[str, str, str, int]:
    raw_name = (upload.filename or "").strip().replace("\\", "/")
    original_name = Path(raw_name).name[:255]
    if not original_name:
        raise HTTPException(status_code=400, detail="Файл не прикреплён")

    extension = Path(original_name).suffix.lower()
    if extension not in ALLOWED_ATTACHMENT_EXTENSIONS:
        raise HTTPException(status_code=400, detail="Недопустимый тип файла")

    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    storage_name = f"{uuid4().hex}{extension}"
    storage_path = UPLOAD_DIR / storage_name
    size = 0

    try:
        with storage_path.open("wb") as destination:
            while chunk := await upload.read(1024 * 1024):
                size += len(chunk)
                if size > MAX_ATTACHMENT_SIZE:
                    raise HTTPException(
                        status_code=413,
                        detail="Файл слишком большой",
                    )
                destination.write(chunk)
    except Exception:
        storage_path.unlink(missing_ok=True)
        raise
    finally:
        await upload.close()

    content_type = (upload.content_type or "application/octet-stream")[:255]
    return original_name, storage_name, content_type, size


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
    entry = _build_entry(payload, db, user)
    db.add(entry)
    db.flush()
    _notify_entry_submitted(db, user, entry)
    db.commit()
    db.refresh(entry)
    return entry_to_out(entry)


@router.post("/entries/with-attachment", response_model=schemas.EntryOut)
async def create_entry_with_attachment(
    catalog_id: int = Form(...),
    officer_id: Optional[int] = Form(None),
    quantity: float = Form(1),
    entry_date: Optional[date] = Form(None, alias="date"),
    comments: str = Form(""),
    attachment: UploadFile = File(...),
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    payload = schemas.EntryCreate(
        catalog_id=catalog_id,
        officer_id=officer_id,
        quantity=quantity,
        date=entry_date,
        comments=comments,
    )
    entry = _build_entry(payload, db, user)
    original_name, storage_name, content_type, size = await _store_attachment(
        attachment,
    )
    entry.attachment_name = original_name
    entry.attachment_storage_name = storage_name
    entry.attachment_content_type = content_type
    entry.attachment_size = size

    try:
        db.add(entry)
        db.flush()
        _notify_entry_submitted(db, user, entry)
        db.commit()
        db.refresh(entry)
    except Exception:
        db.rollback()
        (UPLOAD_DIR / storage_name).unlink(missing_ok=True)
        raise
    return entry_to_out(entry)


@router.get("/entries/{entry_id}/attachment")
def download_attachment(
    entry_id: int,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    entry = db.get(KpiEntry, entry_id)
    if not entry:
        raise HTTPException(status_code=404, detail="Запись не найдена")
    if user.role != "head" and entry.officer_id != user.id:
        raise HTTPException(status_code=403, detail="Недостаточно прав")
    if not entry.attachment_storage_name or not entry.attachment_name:
        raise HTTPException(status_code=404, detail="Вложение не найдено")

    storage_name = Path(entry.attachment_storage_name).name
    if storage_name != entry.attachment_storage_name:
        raise HTTPException(status_code=404, detail="Вложение не найдено")
    storage_path = UPLOAD_DIR / storage_name
    if not storage_path.is_file():
        raise HTTPException(status_code=404, detail="Вложение не найдено")

    return FileResponse(
        storage_path,
        media_type="application/octet-stream",
        filename=entry.attachment_name,
        content_disposition_type="attachment",
        headers={"X-Content-Type-Options": "nosniff"},
    )


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
    previous_status = entry.status
    entry.status = payload.status
    entry.head_comment = payload.head_comment
    if previous_status != payload.status and payload.status in ("confirmed", "rejected"):
        code = entry.catalog.code if entry.catalog else "KPI"
        is_confirmed = payload.status == "confirmed"
        add_notification(
            db,
            user_id=entry.officer_id,
            kind=f"entry_{payload.status}",
            title="Запись KPI подтверждена" if is_confirmed else "Запись KPI отклонена",
            message=(
                f"Запись {code} подтверждена начальником."
                if is_confirmed
                else f"Запись {code} отклонена начальником."
            ),
            link="/entries",
        )
    db.commit()
    db.refresh(entry)
    return entry_to_out(entry)
