"""Общий рейтинг сотрудников по подтвержденным KPI."""

from sqlalchemy import func
from sqlalchemy.orm import Session

from models import KpiEntry, User


def build_efficiency_map(db: Session) -> dict[int, dict]:
    officers = db.query(User).filter(
        User.role == "officer",
    ).order_by(User.full_name).all()
    confirmed_scores = {
        officer_id: float(score or 0)
        for officer_id, score in db.query(
            KpiEntry.officer_id,
            func.sum(KpiEntry.final_score),
        ).filter(
            KpiEntry.status == "confirmed",
        ).group_by(
            KpiEntry.officer_id,
        ).all()
    }
    ranked = sorted(
        officers,
        key=lambda officer: (
            -confirmed_scores.get(officer.id, 0),
            officer.full_name,
        ),
    )
    total = len(ranked)
    return {
        officer.id: {
            "rank": 1 + sum(
                confirmed_scores.get(other.id, 0)
                > confirmed_scores.get(officer.id, 0)
                for other in ranked
            ),
            "total": total,
            "score": round(confirmed_scores.get(officer.id, 0), 2),
        }
        for officer in ranked
    }
