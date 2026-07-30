"""Личные сообщения между начальником и сотрудниками."""

from datetime import datetime

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import and_, or_
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user
from database import get_db
from models import DirectMessage, KpiEntry, User
from services.efficiency import build_efficiency_map

router = APIRouter(prefix="/api/messenger", tags=["messenger"])


def _get_contact_or_404(db: Session, user: User, contact_id: int) -> User:
    contact = db.get(User, contact_id)
    if not contact:
        raise HTTPException(status_code=404, detail="Пользователь не найден")
    if {user.role, contact.role} != {"head", "officer"}:
        raise HTTPException(status_code=403, detail="Этот диалог недоступен")
    return contact


def _conversation_filter(user_id: int, contact_id: int):
    return or_(
        and_(
            DirectMessage.sender_id == user_id,
            DirectMessage.recipient_id == contact_id,
        ),
        and_(
            DirectMessage.sender_id == contact_id,
            DirectMessage.recipient_id == user_id,
        ),
    )


def _conversation_officer(user: User, contact: User) -> User:
    return user if user.role == "officer" else contact


def _entry_out(item: KpiEntry) -> schemas.MessengerEntryOut:
    return schemas.MessengerEntryOut(
        id=item.id,
        date=item.date,
        code=item.catalog.code if item.catalog else "",
        indicator=item.catalog.indicator if item.catalog else "",
        block=item.catalog.block if item.catalog else "",
        npa=item.catalog.npa if item.catalog else "",
        source_url=item.catalog.source_url if item.catalog else "",
        quantity=item.quantity,
        points_per_unit=item.points_per_unit,
        coeff=item.coeff,
        raw_score=item.raw_score,
        final_score=item.final_score,
        status=item.status,
        comments=item.comments or "",
        head_comment=item.head_comment or "",
    )


def _message_out(item: DirectMessage, user_id: int) -> schemas.MessageOut:
    is_deleted = item.deleted_at is not None
    return schemas.MessageOut(
        id=item.id,
        sender_id=item.sender_id,
        recipient_id=item.recipient_id,
        sender_name=item.sender.full_name if item.sender else "",
        body="" if is_deleted else item.body,
        created_at=item.created_at,
        read_at=item.read_at,
        sent_by_me=item.sender_id == user_id,
        is_deleted=is_deleted,
        entry=_entry_out(item.entry) if item.entry and not is_deleted else None,
    )


@router.get("/contacts", response_model=list[schemas.MessengerContactOut])
def list_contacts(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    target_role = "officer" if user.role == "head" else "head"
    contacts = db.query(User).filter(User.role == target_role).all()
    efficiency = build_efficiency_map(db)
    rows = []
    for contact in contacts:
        last_message = db.query(DirectMessage).filter(
            _conversation_filter(user.id, contact.id),
        ).order_by(
            DirectMessage.created_at.desc(),
            DirectMessage.id.desc(),
        ).first()
        unread_count = db.query(DirectMessage).filter(
            DirectMessage.sender_id == contact.id,
            DirectMessage.recipient_id == user.id,
            DirectMessage.read_at.is_(None),
            DirectMessage.deleted_at.is_(None),
        ).count()
        contact_efficiency = efficiency.get(contact.id)
        rows.append(schemas.MessengerContactOut(
            id=contact.id,
            full_name=contact.full_name,
            rank=contact.rank,
            role=contact.role,
            district_name=contact.district.name if contact.district else "",
            unread_count=unread_count,
            last_message=(
                ""
                if last_message and last_message.deleted_at
                else last_message.body if last_message else ""
            ),
            last_message_at=last_message.created_at if last_message else None,
            last_message_deleted=bool(last_message and last_message.deleted_at),
            efficiency_rank=contact_efficiency["rank"] if contact_efficiency else None,
            efficiency_total=contact_efficiency["total"] if contact_efficiency else None,
            efficiency_score=contact_efficiency["score"] if contact_efficiency else None,
        ))
    rows.sort(key=lambda item: item.full_name)
    rows.sort(
        key=lambda item: item.last_message_at or datetime.min,
        reverse=True,
    )
    return rows


@router.get("/entries/{contact_id}", response_model=list[schemas.MessengerEntryOut])
def list_discussion_entries(
    contact_id: int,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    contact = _get_contact_or_404(db, user, contact_id)
    officer = _conversation_officer(user, contact)
    entries = db.query(KpiEntry).filter(
        KpiEntry.officer_id == officer.id,
    ).order_by(
        KpiEntry.date.desc(),
        KpiEntry.id.desc(),
    ).limit(200).all()
    return [_entry_out(item) for item in entries]


@router.get("/messages/{contact_id}", response_model=list[schemas.MessageOut])
def list_messages(
    contact_id: int,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    _get_contact_or_404(db, user, contact_id)
    items = db.query(DirectMessage).filter(
        _conversation_filter(user.id, contact_id),
    ).order_by(
        DirectMessage.created_at.desc(),
        DirectMessage.id.desc(),
    ).limit(500).all()
    items.reverse()
    unread = [
        item
        for item in items
        if item.recipient_id == user.id and item.read_at is None
    ]
    if unread:
        read_at = datetime.utcnow()
        for item in unread:
            item.read_at = read_at
        db.commit()
    return [_message_out(item, user.id) for item in items]


@router.post("/messages", response_model=schemas.MessageOut)
def send_message(
    payload: schemas.MessageCreate,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    contact = _get_contact_or_404(db, user, payload.recipient_id)
    body = payload.body.strip()
    if not body:
        raise HTTPException(status_code=400, detail="Сообщение не может быть пустым")
    entry = None
    if payload.entry_id is not None:
        entry = db.get(KpiEntry, payload.entry_id)
        officer = _conversation_officer(user, contact)
        if not entry or entry.officer_id != officer.id:
            raise HTTPException(
                status_code=404,
                detail="Запись KPI для этого диалога не найдена",
            )
    item = DirectMessage(
        sender_id=user.id,
        recipient_id=payload.recipient_id,
        entry_id=entry.id if entry else None,
        body=body,
    )
    db.add(item)
    db.commit()
    db.refresh(item)
    return _message_out(item, user.id)


@router.delete("/messages/{message_id}", response_model=schemas.MessageOut)
def delete_message(
    message_id: int,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    item = db.get(DirectMessage, message_id)
    if not item:
        raise HTTPException(status_code=404, detail="Сообщение не найдено")
    if item.sender_id != user.id:
        raise HTTPException(
            status_code=403,
            detail="Можно удалить только свое сообщение",
        )
    if item.deleted_at is None:
        deleted_at = datetime.utcnow()
        item.deleted_at = deleted_at
        if item.read_at is None:
            item.read_at = deleted_at
        db.commit()
        db.refresh(item)
    return _message_out(item, user.id)
