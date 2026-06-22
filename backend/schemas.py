"""Pydantic-схемы для запросов и ответов API."""
from datetime import date as date_type
from typing import Optional

from pydantic import BaseModel, ConfigDict


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


class EntryCreate(BaseModel):
    catalog_id: int
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
