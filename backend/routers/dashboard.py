"""Маршрут агрегированного дашборда."""

from typing import Optional

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from auth import require_head
from database import get_db
from models import User
from services.dashboard import build_dashboard

router = APIRouter(prefix="/api", tags=["dashboard"])


@router.get("/dashboard")
def dashboard(
    month: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    district_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    return build_dashboard(
        db,
        month=month,
        officer_id=officer_id,
        district_id=district_id,
    )
