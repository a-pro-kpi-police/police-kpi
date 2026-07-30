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
        subblock=entry.catalog.subblock if entry.catalog else "",
        npa=entry.catalog.npa if entry.catalog else "",
        source_url=entry.catalog.source_url if entry.catalog else "",
        quantity=entry.quantity,
        counted_quantity=entry.counted_quantity,
        monthly_limit=entry.catalog.monthly_limit if entry.catalog else 0,
        is_mass=entry.catalog.is_mass if entry.catalog else False,
        is_critical=entry.catalog.is_critical if entry.catalog else False,
        points_per_unit=entry.points_per_unit,
        coeff=entry.coeff,
        raw_score=entry.raw_score,
        capped_score=entry.capped_score,
        anti_coeff=entry.anti_coeff,
        final_score=entry.final_score,
        status=entry.status,
        comments=entry.comments or "",
        head_comment=entry.head_comment or "",
        attachment_name=entry.attachment_name or "",
        attachment_content_type=entry.attachment_content_type or "",
        attachment_size=entry.attachment_size or 0,
    )
