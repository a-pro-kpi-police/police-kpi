"""Маршруты справочников и доступных периодов."""

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user, require_head
from database import get_db
from models import District, KpiCatalog, KpiEntry, User

router = APIRouter(prefix="/api", tags=["catalog"])


@router.get("/catalog", response_model=list[schemas.CatalogOut])
def get_catalog(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    query = db.query(KpiCatalog)
    if user.role != "head":
        query = query.filter(KpiCatalog.points_per_unit >= 0)
    return query.order_by(KpiCatalog.code).all()


@router.get("/districts", response_model=list[schemas.DistrictOut])
def get_districts(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    return db.query(District).all()


@router.get("/officers", response_model=list[schemas.UserOut])
def list_officers(
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    return db.query(User).filter(
        User.role == "officer",
    ).order_by(User.full_name).all()


@router.get("/months")
def months(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    query = db.query(KpiEntry.month).distinct()
    if user.role != "head":
        query = query.filter(KpiEntry.officer_id == user.id)
    return sorted({month for (month,) in query.all()}, reverse=True)
