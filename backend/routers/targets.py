"""Маршруты индивидуальных и групповых целей KPI."""

from datetime import date
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session

import schemas
from auth import get_current_user, require_head
from database import get_db
from models import User
from services.targets import (
    apply_target_quantities,
    build_group_target_rows,
    build_target_overview,
    build_target_progress,
    build_target_rows,
    get_district_or_404,
    get_officer_or_404,
    target_quantities_or_404,
    validate_month,
)

router = APIRouter(prefix="/api", tags=["targets"])


@router.get("/targets", response_model=list[schemas.TargetOut])
def get_targets(
    month: str = Query(...),
    officer_id: int = Query(...),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    officer = get_officer_or_404(db, officer_id)
    return build_target_rows(db, officer, month)


@router.put("/targets", response_model=list[schemas.TargetOut])
def save_targets(
    payload: schemas.TargetSavePayload,
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    officer = get_officer_or_404(db, payload.officer_id)
    validate_month(payload.month)
    quantities = target_quantities_or_404(db, payload.items)
    apply_target_quantities(
        db,
        [officer.id],
        payload.month,
        quantities,
        replace=True,
    )
    db.commit()
    return build_target_rows(db, officer, payload.month)


@router.get("/group-targets", response_model=list[schemas.GroupTargetOut])
def get_group_targets(
    month: str = Query(...),
    district_id: int = Query(...),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    district = get_district_or_404(db, district_id)
    return build_group_target_rows(db, district, month)


@router.put("/group-targets", response_model=list[schemas.GroupTargetOut])
def save_group_targets(
    payload: schemas.GroupTargetSavePayload,
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    district = get_district_or_404(db, payload.district_id)
    validate_month(payload.month)
    officer_ids = [
        officer_id
        for (officer_id,) in db.query(User.id).filter(
            User.role == "officer",
            User.district_id == district.id,
        ).all()
    ]
    if not officer_ids:
        raise HTTPException(status_code=400, detail="В группе нет сотрудников")

    quantities = target_quantities_or_404(db, payload.items)
    apply_target_quantities(
        db,
        officer_ids,
        payload.month,
        quantities,
        replace=False,
    )
    db.commit()
    return build_group_target_rows(db, district, payload.month)


@router.get("/target-progress", response_model=schemas.TargetProgressOut)
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


@router.get("/target-overview", response_model=schemas.TargetOverviewOut)
def target_overview(
    month: Optional[str] = Query(None),
    officer_id: Optional[int] = Query(None),
    db: Session = Depends(get_db),
    head: User = Depends(require_head),
):
    selected_month = month or date.today().strftime("%Y-%m")
    return build_target_overview(db, selected_month, officer_id)
