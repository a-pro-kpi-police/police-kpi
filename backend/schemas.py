"""Pydantic-схемы для запросов и ответов API."""
from datetime import date as date_type
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field


class Token(BaseModel):
    access_token: str
    token_type: str = "bearer"


class DistrictOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: int
    name: str
    coeff: float


class UserOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: int
    username: str
    full_name: str
    rank: str
    role: str
    district: Optional[DistrictOut] = None


class CatalogOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: int
    code: str
    indicator: str
    block: str
    npa: str
    points_per_unit: float
    unit: str


class TargetOut(BaseModel):
    catalog_id: int
    code: str
    indicator: str
    unit: str
    points_per_unit: float
    target_quantity: float
    target_score: float


class TargetSaveItem(BaseModel):
    catalog_id: int
    target_quantity: float = Field(ge=0)


class TargetSavePayload(BaseModel):
    officer_id: int
    month: str
    items: list[TargetSaveItem]


class GroupTargetOut(TargetOut):
    mixed: bool


class GroupTargetSavePayload(BaseModel):
    district_id: int
    month: str
    items: list[TargetSaveItem]


class TargetProgressItemOut(BaseModel):
    catalog_id: int
    code: str
    indicator: str
    unit: str
    points_per_unit: float
    target_quantity: float
    confirmed_quantity: float
    remaining_quantity: float
    target_score: float
    confirmed_score: float
    achievement_percent: float


class TargetProgressOut(BaseModel):
    month: str
    officer_id: int
    actual_score: float
    target_score: float
    achievement_percent: float
    items: list[TargetProgressItemOut]


class TargetOverviewOfficerOut(BaseModel):
    officer_id: int
    name: str
    actual_score: float
    target_score: float
    achievement_percent: float
    gap: float
    status: str


class TargetOverviewOut(BaseModel):
    month: str
    actual_score: float
    target_score: float
    achievement_percent: float
    on_target_count: int
    officers_count: int
    officers: list[TargetOverviewOfficerOut]
    items: list[TargetProgressItemOut]


class EntryCreate(BaseModel):
    catalog_id: int
    officer_id: Optional[int] = None
    quantity: float = 1
    date: Optional[date_type] = None
    comments: str = ""


class EntryStatusUpdate(BaseModel):
    status: str  # confirmed | rejected | pending
    head_comment: str = ""


class EntryOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: int
    date: date_type
    month: str
    officer_id: int
    officer_name: str
    rank: str
    district_name: str
    code: str
    indicator: str
    block: str
    npa: str
    quantity: float
    points_per_unit: float
    coeff: float
    raw_score: float
    final_score: float
    status: str
    comments: str
    head_comment: str
