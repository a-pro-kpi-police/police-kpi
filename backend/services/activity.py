"""Создание системных уведомлений без отдельных транзакций."""

from sqlalchemy.orm import Session

from models import AppNotification, User


def add_notification(
    db: Session,
    user_id: int,
    kind: str,
    title: str,
    message: str,
    link: str,
) -> AppNotification:
    notification = AppNotification(
        user_id=user_id,
        kind=kind,
        title=title,
        message=message,
        link=link,
    )
    db.add(notification)
    return notification


def notify_heads(
    db: Session,
    kind: str,
    title: str,
    message: str,
    link: str,
) -> None:
    head_ids = [
        head_id
        for (head_id,) in db.query(User.id).filter(User.role == "head").all()
    ]
    for head_id in head_ids:
        add_notification(db, head_id, kind, title, message, link)
