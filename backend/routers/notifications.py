"""Маршруты персональных системных уведомлений."""

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user
from database import get_db
from models import AppNotification, User

router = APIRouter(prefix="/api/notifications", tags=["notifications"])


def _notification_out(item: AppNotification) -> schemas.NotificationOut:
    return schemas.NotificationOut(
        id=item.id,
        kind=item.kind,
        title=item.title,
        message=item.message,
        link=item.link,
        is_read=item.is_read,
        created_at=item.created_at,
    )


@router.get("", response_model=schemas.NotificationListOut)
def list_notifications(
    limit: int = Query(20, ge=1, le=100),
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    query = db.query(AppNotification).filter(
        AppNotification.user_id == user.id,
    )
    unread_count = query.filter(AppNotification.is_read.is_(False)).count()
    items = query.order_by(
        AppNotification.created_at.desc(),
        AppNotification.id.desc(),
    ).limit(limit).all()
    return schemas.NotificationListOut(
        unread_count=unread_count,
        items=[_notification_out(item) for item in items],
    )


@router.patch("/{notification_id}/read", response_model=schemas.NotificationOut)
def mark_notification_read(
    notification_id: int,
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    item = db.query(AppNotification).filter(
        AppNotification.id == notification_id,
        AppNotification.user_id == user.id,
    ).first()
    if not item:
        raise HTTPException(status_code=404, detail="Уведомление не найдено")
    item.is_read = True
    db.commit()
    db.refresh(item)
    return _notification_out(item)


@router.post("/read-all")
def mark_all_notifications_read(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    updated = db.query(AppNotification).filter(
        AppNotification.user_id == user.id,
        AppNotification.is_read.is_(False),
    ).update({AppNotification.is_read: True}, synchronize_session=False)
    db.commit()
    return {"updated": updated}
