"""Workbook-compatible monthly KPI scoring."""

from collections import defaultdict

from sqlalchemy.orm import Session

from models import KpiEntry


MASS_THRESHOLD_1 = 0.40
MASS_THRESHOLD_2 = 0.60
MASS_COEFFICIENT_1 = 0.85
MASS_COEFFICIENT_2 = 0.70


def mass_coefficient(mass_share: float) -> float:
    if mass_share > MASS_THRESHOLD_2:
        return MASS_COEFFICIENT_2
    if mass_share > MASS_THRESHOLD_1:
        return MASS_COEFFICIENT_1
    return 1.0


def _entry_limit(entry: KpiEntry) -> float:
    if entry.catalog and entry.catalog.monthly_limit > 0:
        return float(entry.catalog.monthly_limit)
    return float("inf")


def recalculate_officer_month(
    db: Session,
    officer_id: int,
    month: str,
) -> dict:
    """Recalculate all entries for one officer and month.

    Confirmed quantities are capped per KPI code. The mass-indicator
    coefficient is calculated from capped positive scores, and any confirmed
    critical indicator zeroes the complete monthly result.
    """
    entries = (
        db.query(KpiEntry)
        .filter(
            KpiEntry.officer_id == officer_id,
            KpiEntry.month == month,
        )
        .order_by(KpiEntry.date, KpiEntry.id)
        .all()
    )
    confirmed = [entry for entry in entries if entry.status == "confirmed"]
    by_catalog = defaultdict(list)
    for entry in confirmed:
        by_catalog[entry.catalog_id].append(entry)

    for entry in entries:
        entry.raw_score = round(entry.quantity * entry.points_per_unit, 2)
        if entry.status != "confirmed":
            preview_quantity = min(
                max(float(entry.quantity), 0.0),
                _entry_limit(entry),
            )
            entry.counted_quantity = round(preview_quantity, 2)
            entry.capped_score = round(
                preview_quantity * entry.points_per_unit,
                2,
            )
            entry.anti_coeff = 1.0
            entry.final_score = round(entry.capped_score * entry.coeff, 2)

    for catalog_entries in by_catalog.values():
        remaining = _entry_limit(catalog_entries[0])
        for entry in catalog_entries:
            quantity = max(float(entry.quantity), 0.0)
            counted = quantity if remaining == float("inf") else min(
                quantity,
                max(remaining, 0.0),
            )
            entry.counted_quantity = round(counted, 2)
            entry.capped_score = round(
                entry.counted_quantity * entry.points_per_unit,
                2,
            )
            if remaining != float("inf"):
                remaining -= counted

    positive_score = round(sum(
        entry.capped_score
        for entry in confirmed
        if entry.points_per_unit >= 0
    ), 2)
    negative_score = round(sum(
        entry.capped_score
        for entry in confirmed
        if entry.points_per_unit < 0
    ), 2)
    mass_score = round(sum(
        entry.capped_score
        for entry in confirmed
        if (
            entry.points_per_unit >= 0
            and entry.catalog
            and entry.catalog.is_mass
        )
    ), 2)
    mass_share = mass_score / positive_score if positive_score else 0.0
    anti = mass_coefficient(mass_share)
    critical = any(
        entry.quantity > 0
        and entry.catalog
        and entry.catalog.is_critical
        for entry in confirmed
    )
    month_coeff = float(confirmed[0].coeff) if confirmed else 1.0

    for entry in confirmed:
        entry.coeff = month_coeff
        entry.anti_coeff = anti

    adjusted_positive = positive_score * anti * month_coeff
    target_final = (
        0.0
        if critical
        else round(max(0.0, adjusted_positive + negative_score), 2)
    )

    if critical:
        for entry in confirmed:
            entry.final_score = 0.0
    else:
        penalty_scale = 1.0
        if negative_score < 0 and adjusted_positive + negative_score < 0:
            penalty_scale = (
                adjusted_positive / abs(negative_score)
                if adjusted_positive > 0
                else 0.0
            )
        scored_entries = []
        for entry in confirmed:
            if entry.points_per_unit >= 0:
                final_score = entry.capped_score * anti * month_coeff
            else:
                final_score = entry.capped_score * penalty_scale
            entry.final_score = round(final_score, 2)
            if entry.counted_quantity > 0:
                scored_entries.append(entry)

        if scored_entries:
            allocated = round(sum(
                entry.final_score for entry in confirmed
            ), 2)
            adjustment = round(target_final - allocated, 2)
            scored_entries[-1].final_score = round(
                scored_entries[-1].final_score + adjustment,
                2,
            )

    db.flush()
    return {
        "officer_id": officer_id,
        "month": month,
        "reported_quantity": round(sum(
            entry.quantity for entry in confirmed
        ), 2),
        "counted_quantity": round(sum(
            entry.counted_quantity for entry in confirmed
        ), 2),
        "positive_score": positive_score,
        "negative_score": negative_score,
        "mass_score": mass_score,
        "mass_share": round(mass_share, 6),
        "anti_coeff": anti,
        "condition_coeff": month_coeff,
        "critical": critical,
        "final_score": target_final,
    }
