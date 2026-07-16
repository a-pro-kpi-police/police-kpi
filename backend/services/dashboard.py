"""Формирование агрегатов для дашборда начальника отдела."""

from collections import defaultdict
from typing import Optional

from sqlalchemy.orm import Session

from models import KpiEntry


def build_dashboard(
    db: Session,
    month: Optional[str] = None,
    officer_id: Optional[int] = None,
) -> dict:
    query = db.query(KpiEntry)
    if officer_id:
        query = query.filter(KpiEntry.officer_id == officer_id)
    if month:
        query = query.filter(KpiEntry.month == month)
    entries = query.all()

    confirmed = [entry for entry in entries if entry.status == "confirmed"]
    pending = [entry for entry in entries if entry.status == "pending"]
    rejected = [entry for entry in entries if entry.status == "rejected"]

    total_final = round(sum(entry.final_score for entry in confirmed), 2)
    by_officer = defaultdict(lambda: {
        "final": 0.0,
        "raw": 0.0,
        "confirmed": 0,
        "pending": 0,
        "name": "",
        "rank": "",
        "district": "",
        "blocks": defaultdict(float),
    })
    for entry in entries:
        officer = by_officer[entry.officer_id]
        officer["name"] = entry.officer.full_name if entry.officer else ""
        officer["rank"] = entry.officer.rank if entry.officer else ""
        officer["district"] = (
            entry.officer.district.name
            if entry.officer and entry.officer.district
            else ""
        )
        if entry.status == "confirmed":
            officer["final"] += entry.final_score
            officer["raw"] += entry.raw_score
            officer["confirmed"] += 1
            block = entry.catalog.block if entry.catalog else "—"
            officer["blocks"][block] += entry.final_score
        elif entry.status == "pending":
            officer["pending"] += 1

    officers = [
        {
            "officer_id": officer_id,
            "name": data["name"],
            "rank": data["rank"],
            "district": data["district"],
            "final": round(data["final"], 2),
            "raw": round(data["raw"], 2),
            "confirmed": data["confirmed"],
            "pending": data["pending"],
            "blocks": {
                block: round(value, 2)
                for block, value in data["blocks"].items()
            },
        }
        for officer_id, data in by_officer.items()
    ]
    officers.sort(key=lambda item: item["final"], reverse=True)

    by_block = defaultdict(float)
    for entry in confirmed:
        block = entry.catalog.block if entry.catalog else "—"
        by_block[block] += entry.final_score
    blocks = [
        {"block": block, "final": round(value, 2)}
        for block, value in sorted(by_block.items(), key=lambda item: -item[1])
    ]

    by_month = defaultdict(float)
    for entry in confirmed:
        by_month[entry.month] += entry.final_score
    trend = [
        {"month": item_month, "final": round(value, 2)}
        for item_month, value in sorted(by_month.items())
    ]

    by_month_status = defaultdict(
        lambda: {"confirmed": 0, "pending": 0, "rejected": 0},
    )
    for entry in entries:
        by_month_status[entry.month][entry.status] += 1
    status_trend = [
        {"month": item_month, **counts}
        for item_month, counts in sorted(by_month_status.items())
    ]

    by_indicator = defaultdict(
        lambda: {"quantity": 0.0, "code": "", "indicator": ""},
    )
    for entry in entries:
        code = entry.catalog.code if entry.catalog else "—"
        item = by_indicator[code]
        item["code"] = code
        item["indicator"] = entry.catalog.indicator if entry.catalog else "—"
        item["quantity"] += entry.quantity
    indicator_volume = [
        {**item, "quantity": round(item["quantity"], 2)}
        for item in sorted(
            by_indicator.values(),
            key=lambda value: -value["quantity"],
        )
    ]

    by_district_review = defaultdict(lambda: {"pending": 0, "rejected": 0})
    for entry in pending + rejected:
        district = (
            entry.officer.district.name
            if entry.officer and entry.officer.district
            else "—"
        )
        by_district_review[district][entry.status] += 1
    district_review = [
        {"district": district, **counts}
        for district, counts in sorted(
            by_district_review.items(),
            key=lambda item: -(item[1]["pending"] + item[1]["rejected"]),
        )
    ]

    total_decided = len(confirmed) + len(rejected)
    confirm_rate = (
        round(100 * len(confirmed) / total_decided, 1)
        if total_decided
        else 0.0
    )
    return {
        "total_final": total_final,
        "confirmed_count": len(confirmed),
        "pending_count": len(pending),
        "rejected_count": len(rejected),
        "officers_count": len(officers),
        "confirm_rate": confirm_rate,
        "officers": officers,
        "blocks": blocks,
        "trend": trend,
        "status_trend": status_trend,
        "indicator_volume": indicator_volume,
        "district_review": district_review,
    }
