"""Pydantic-схемы для запросов и ответов API."""
from datetime import date as date_type, datetime
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


class OfficerOut(UserOut):
    efficiency_rank: int
    efficiency_total: int
    efficiency_score: float


class CatalogOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)
    id: int
    code: str
    indicator: str
    block: str
    subblock: str
    npa: str
    points_per_unit: float
    unit: str
    source_url: str


class TargetOut(BaseModel):
    catalog_id: int
    code: str
    indicator: str
    block: str
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
    block: str
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
    penalty_score: float
    penalty_count: int


class TargetBlockProgressOut(BaseModel):
    block: str
    actual_score: float
    target_score: float
    achievement_percent: float


class TargetOverviewOut(BaseModel):
    month: str
    actual_score: float
    target_score: float
    achievement_percent: float
    on_target_count: int
    officers_count: int
    penalty_score: float
    penalty_count: int
    officers: list[TargetOverviewOfficerOut]
    blocks: list[TargetBlockProgressOut]
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
    subblock: str
    npa: str
    source_url: str
    quantity: float
    points_per_unit: float
    coeff: float
    raw_score: float
    final_score: float
    status: str
    comments: str
    head_comment: str
    attachment_name: str
    attachment_content_type: str
    attachment_size: int


class NotificationOut(BaseModel):
    id: int
    kind: str
    title: str
    message: str
    link: str
    is_read: bool
    created_at: datetime


class NotificationListOut(BaseModel):
    unread_count: int
    items: list[NotificationOut]


class MessengerContactOut(BaseModel):
    id: int
    full_name: str
    rank: str
    role: str
    district_name: str
    unread_count: int
    last_message: str
    last_message_at: Optional[datetime] = None
    last_message_deleted: bool
    efficiency_rank: Optional[int] = None
    efficiency_total: Optional[int] = None
    efficiency_score: Optional[float] = None


class MessageCreate(BaseModel):
    recipient_id: int
    body: str = Field(min_length=1, max_length=2000)
    entry_id: Optional[int] = None


class MessengerEntryOut(BaseModel):
    id: int
    date: date_type
    code: str
    indicator: str
    block: str
    npa: str
    source_url: str
    quantity: float
    points_per_unit: float
    coeff: float
    raw_score: float
    final_score: float
    status: str
    comments: str
    head_comment: str


class MessageOut(BaseModel):
    id: int
    sender_id: int
    recipient_id: int
    sender_name: str
    body: str
    created_at: datetime
    read_at: Optional[datetime] = None
    sent_by_me: bool
    is_deleted: bool
    entry: Optional[MessengerEntryOut] = None
