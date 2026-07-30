"""Расчёт и назначение индивидуальных и групповых целей KPI."""

from collections import defaultdict
from typing import Optional

from fastapi import HTTPException
from sqlalchemy.orm import Session

import schemas
from models import District, KpiCatalog, KpiEntry, KpiTarget, User


def _achievement_percent(actual_score: float, target_score: float) -> float:
    """Return monthly norm attainment, allowing results above 100%."""
    if not target_score:
        return 0.0
    return round(max(0.0, 100 * actual_score / target_score), 1)


def validate_month(month: str):
    if not isinstance(month, str) or len(month) != 7 or month[4] != "-":
        raise HTTPException(status_code=400, detail="Месяц указан неверно")
    year, month_num = month.split("-", 1)
    if not year.isdigit() or not month_num.isdigit() or not (1 <= int(month_num) <= 12):
        raise HTTPException(status_code=400, detail="Месяц указан неверно")


def get_officer_or_404(db: Session, officer_id: int) -> User:
    officer = db.get(User, officer_id)
    if not officer or officer.role != "officer":
        raise HTTPException(status_code=404, detail="Сотрудник не найден")
    return officer


def get_district_or_404(db: Session, district_id: int) -> District:
    district = db.get(District, district_id)
    if not district:
        raise HTTPException(status_code=404, detail="Группа не найдена")
    return district


def target_quantities_or_404(
    db: Session,
    items: list[schemas.TargetSaveItem],
) -> dict[int, float]:
    catalog_ids = {
        catalog_id
        for (catalog_id,) in db.query(KpiCatalog.id).filter(
            KpiCatalog.is_active.is_(True),
            KpiCatalog.points_per_unit >= 0,
        ).all()
    }
    quantities = {}
    for item in items:
        if item.catalog_id not in catalog_ids:
            raise HTTPException(status_code=404, detail="Показатель KPI не найден")
        quantities[item.catalog_id] = item.target_quantity
    return quantities


def apply_target_quantities(
    db: Session,
    officer_ids: list[int],
    month: str,
    quantities: dict[int, float],
    replace: bool,
):
    existing = (
        db.query(KpiTarget).filter(
            KpiTarget.officer_id.in_(officer_ids),
            KpiTarget.month == month,
        ).all()
        if officer_ids
        else []
    )
    existing_by_key = {
        (target.officer_id, target.catalog_id): target
        for target in existing
    }

    for target in existing:
        should_delete = (
            (replace and target.catalog_id not in quantities)
            or (target.catalog_id in quantities and quantities[target.catalog_id] <= 0)
        )
        if should_delete:
            db.delete(target)

    for officer_id in officer_ids:
        for catalog_id, target_quantity in quantities.items():
            if target_quantity <= 0:
                continue
            current = existing_by_key.get((officer_id, catalog_id))
            if current:
                current.target_quantity = target_quantity
            else:
                db.add(KpiTarget(
                    month=month,
                    officer_id=officer_id,
                    catalog_id=catalog_id,
                    target_quantity=target_quantity,
                ))


def build_target_rows(
    db: Session,
    officer: User,
    month: str,
) -> list[schemas.TargetOut]:
    validate_month(month)
    coeff = officer.district.coeff if officer.district else 1.0
    targets = db.query(KpiTarget).filter(
        KpiTarget.officer_id == officer.id,
        KpiTarget.month == month,
    ).all()
    quantities = {target.catalog_id: target.target_quantity for target in targets}

    rows = []
    catalog = db.query(KpiCatalog).filter(
        KpiCatalog.is_active.is_(True),
        KpiCatalog.points_per_unit >= 0,
    ).order_by(KpiCatalog.block, KpiCatalog.code).all()
    for item in catalog:
        target_quantity = round(quantities.get(item.id, 0), 2)
        rows.append(schemas.TargetOut(
            catalog_id=item.id,
            code=item.code,
            indicator=item.indicator,
            block=item.block,
            unit=item.unit,
            points_per_unit=item.points_per_unit,
            monthly_limit=item.monthly_limit,
            is_mass=item.is_mass,
            is_critical=item.is_critical,
            target_quantity=target_quantity,
            target_score=round(target_quantity * item.points_per_unit * coeff, 2),
        ))
    return rows


def build_group_target_rows(
    db: Session,
    district: District,
    month: str,
) -> list[schemas.GroupTargetOut]:
    validate_month(month)
    officer_ids = [
        officer_id
        for (officer_id,) in db.query(User.id).filter(
            User.role == "officer",
            User.district_id == district.id,
        ).all()
    ]
    targets = (
        db.query(KpiTarget).filter(
            KpiTarget.officer_id.in_(officer_ids),
            KpiTarget.month == month,
        ).all()
        if officer_ids
        else []
    )
    quantities = {
        (target.officer_id, target.catalog_id): round(target.target_quantity, 2)
        for target in targets
    }

    rows = []
    catalog = db.query(KpiCatalog).filter(
        KpiCatalog.is_active.is_(True),
        KpiCatalog.points_per_unit >= 0,
    ).order_by(KpiCatalog.block, KpiCatalog.code).all()
    for item in catalog:
        values = [
            quantities.get((officer_id, item.id), 0.0)
            for officer_id in officer_ids
        ]
        mixed = len(set(values)) > 1
        target_quantity = 0.0 if mixed or not values else values[0]
        rows.append(schemas.GroupTargetOut(
            catalog_id=item.id,
            code=item.code,
            indicator=item.indicator,
            block=item.block,
            unit=item.unit,
            points_per_unit=item.points_per_unit,
            monthly_limit=item.monthly_limit,
            is_mass=item.is_mass,
            is_critical=item.is_critical,
            target_quantity=target_quantity,
            target_score=round(
                target_quantity * item.points_per_unit * district.coeff,
                2,
            ),
            mixed=mixed,
        ))
    return rows


def build_target_progress(
    db: Session,
    officer: User,
    month: str,
) -> schemas.TargetProgressOut:
    rows = build_target_rows(db, officer, month)
    confirmed_entries = db.query(KpiEntry).filter(
        KpiEntry.officer_id == officer.id,
        KpiEntry.month == month,
        KpiEntry.status == "confirmed",
    ).all()

    actual_by_catalog = defaultdict(
        lambda: {"quantity": 0.0, "counted_quantity": 0.0, "score": 0.0},
    )
    for entry in confirmed_entries:
        actual_by_catalog[entry.catalog_id]["quantity"] += entry.quantity
        actual_by_catalog[entry.catalog_id]["counted_quantity"] += (
            entry.counted_quantity
        )
        actual_by_catalog[entry.catalog_id]["score"] += entry.final_score

    items = []
    actual_total = 0.0
    for row in rows:
        actual = actual_by_catalog[row.catalog_id]
        confirmed_quantity = round(actual["quantity"], 2)
        counted_quantity = round(actual["counted_quantity"], 2)
        confirmed_score = round(actual["score"], 2)
        remaining_quantity = round(
            max(row.target_quantity - confirmed_quantity, 0),
            2,
        )
        achievement_percent = _achievement_percent(
            confirmed_score,
            row.target_score,
        )
        items.append(schemas.TargetProgressItemOut(
            catalog_id=row.catalog_id,
            code=row.code,
            indicator=row.indicator,
            block=row.block,
            unit=row.unit,
            points_per_unit=row.points_per_unit,
            monthly_limit=row.monthly_limit,
            is_mass=row.is_mass,
            is_critical=row.is_critical,
            target_quantity=row.target_quantity,
            confirmed_quantity=confirmed_quantity,
            counted_quantity=counted_quantity,
            remaining_quantity=remaining_quantity,
            target_score=row.target_score,
            confirmed_score=confirmed_score,
            achievement_percent=achievement_percent,
        ))
        actual_total += confirmed_score

    actual_total = round(actual_total, 2)
    target_total = round(officer.monthly_norm or 100.0, 2)
    return schemas.TargetProgressOut(
        month=month,
        officer_id=officer.id,
        actual_score=actual_total,
        target_score=target_total,
        achievement_percent=_achievement_percent(actual_total, target_total),
        items=items,
    )


def build_target_overview(
    db: Session,
    month: str,
    officer_id: Optional[int] = None,
    district_id: Optional[int] = None,
) -> schemas.TargetOverviewOut:
    validate_month(month)
    if district_id:
        get_district_or_404(db, district_id)
    officers = (
        [get_officer_or_404(db, officer_id)]
        if officer_id
        else db.query(User).filter(
            User.role == "officer",
            *([User.district_id == district_id] if district_id else []),
        ).order_by(User.full_name).all()
    )

    officer_ids = [officer.id for officer in officers]
    penalty_entries = (
        db.query(KpiEntry).filter(
            KpiEntry.officer_id.in_(officer_ids),
            KpiEntry.month == month,
            KpiEntry.status == "confirmed",
            KpiEntry.points_per_unit < 0,
            KpiEntry.quantity > 0,
        ).all()
        if officer_ids
        else []
    )
    penalty_by_officer = defaultdict(lambda: {"score": 0.0, "count": 0})
    for entry in penalty_entries:
        penalty = penalty_by_officer[entry.officer_id]
        penalty["score"] += entry.final_score
        penalty["count"] += 1

    rows = []
    items = []
    actual_total = 0.0
    target_total = 0.0
    penalty_total = 0.0
    penalty_count = 0
    on_target_count = 0
    by_block = defaultdict(lambda: {"actual_score": 0.0, "target_score": 0.0})

    for officer in officers:
        progress = build_target_progress(db, officer, month)
        penalty = penalty_by_officer[officer.id]
        gap = round(progress.actual_score - progress.target_score, 2)
        status = (
            "no_target"
            if progress.target_score == 0
            else "on_target"
            if progress.actual_score >= progress.target_score
            else "behind"
        )
        if status == "on_target":
            on_target_count += 1
        rows.append(schemas.TargetOverviewOfficerOut(
            officer_id=officer.id,
            name=officer.full_name,
            actual_score=progress.actual_score,
            target_score=progress.target_score,
            achievement_percent=progress.achievement_percent,
            gap=gap,
            status=status,
            penalty_score=round(penalty["score"], 2),
            penalty_count=penalty["count"],
        ))
        actual_total += progress.actual_score
        target_total += progress.target_score
        penalty_total += penalty["score"]
        penalty_count += penalty["count"]
        if officer_id:
            items = progress.items
        for item in progress.items:
            block = by_block[item.block]
            block["actual_score"] += item.confirmed_score
            block["target_score"] += item.target_score

    if not officer_id:
        status_order = {"behind": 0, "no_target": 1, "on_target": 2}
        rows.sort(key=lambda row: (
            status_order.get(row.status, 9),
            row.achievement_percent if row.target_score else 9999,
            row.gap,
            row.name,
        ))

    actual_total = round(actual_total, 2)
    target_total = round(target_total, 2)
    blocks = [
        schemas.TargetBlockProgressOut(
            block=block_name,
            actual_score=round(values["actual_score"], 2),
            target_score=round(values["target_score"], 2),
            achievement_percent=_achievement_percent(
                values["actual_score"],
                values["target_score"],
            ),
        )
        for block_name, values in by_block.items()
        if values["target_score"] > 0
    ]
    return schemas.TargetOverviewOut(
        month=month,
        actual_score=actual_total,
        target_score=target_total,
        achievement_percent=_achievement_percent(actual_total, target_total),
        on_target_count=on_target_count,
        officers_count=len(rows),
        penalty_score=round(penalty_total, 2),
        penalty_count=penalty_count,
        officers=rows,
        blocks=blocks,
        items=items,
    )
