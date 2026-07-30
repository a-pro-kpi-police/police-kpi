"""Маршруты справочников и доступных периодов."""

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user, require_head
from database import get_db
from models import District, KpiCatalog, KpiEntry, KpiTarget, User
from services.efficiency import build_efficiency_map

router = APIRouter(prefix="/api", tags=["catalog"])


@router.get("/catalog", response_model=list[schemas.CatalogOut])
def get_catalog(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    query = db.query(KpiCatalog).filter(KpiCatalog.is_active.is_(True))
    if user.role != "head":
        query = query.filter(KpiCatalog.points_per_unit >= 0)
    return query.order_by(KpiCatalog.block, KpiCatalog.code).all()


@router.get("/districts", response_model=list[schemas.DistrictOut])
def get_districts(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    return db.query(District).all()


@router.get("/officers", response_model=list[schemas.OfficerOut])
def list_officers(
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    officers = db.query(User).filter(
        User.role == "officer",
    ).order_by(User.full_name).all()
    efficiency = build_efficiency_map(db)
    return [
        {
            "id": officer.id,
            "username": officer.username,
            "full_name": officer.full_name,
            "rank": officer.rank,
            "role": officer.role,
            "district": officer.district,
            "efficiency_rank": efficiency[officer.id]["rank"],
            "efficiency_total": efficiency[officer.id]["total"],
            "efficiency_score": efficiency[officer.id]["score"],
        }
        for officer in officers
    ]


@router.get("/months")
def months(
    db: Session = Depends(get_db),
    user: User = Depends(get_current_user),
):
    entry_query = db.query(KpiEntry.month).distinct()
    target_query = db.query(KpiTarget.month).distinct()
    if user.role != "head":
        entry_query = entry_query.filter(KpiEntry.officer_id == user.id)
        target_query = target_query.filter(KpiTarget.officer_id == user.id)
    months_with_entries = {month for (month,) in entry_query.all()}
    months_with_targets = {month for (month,) in target_query.all()}
    return sorted(months_with_entries | months_with_targets, reverse=True)
