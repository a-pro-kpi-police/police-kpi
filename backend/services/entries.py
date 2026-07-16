"""Преобразование записей KPI в API-схемы."""

import schemas
from models import KpiEntry


def entry_to_out(entry: KpiEntry) -> schemas.EntryOut:
    return schemas.EntryOut(
        id=entry.id,
        date=entry.date,
        month=entry.month,
        officer_id=entry.officer_id,
        officer_name=entry.officer.full_name if entry.officer else "",
        rank=entry.officer.rank if entry.officer else "",
        district_name=(
            entry.officer.district.name
            if entry.officer and entry.officer.district
            else ""
        ),
        code=entry.catalog.code if entry.catalog else "",
        indicator=entry.catalog.indicator if entry.catalog else "",
        block=entry.catalog.block if entry.catalog else "",
        npa=entry.catalog.npa if entry.catalog else "",
        quantity=entry.quantity,
        points_per_unit=entry.points_per_unit,
        coeff=entry.coeff,
        raw_score=entry.raw_score,
        final_score=entry.final_score,
        status=entry.status,
        comments=entry.comments or "",
        head_comment=entry.head_comment or "",
    )
