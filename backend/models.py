"""Модели данных: пользователи, каталог KPI, участки и записи KPI."""
from datetime import datetime

from sqlalchemy import Boolean, Column, Integer, String, Float, Date, DateTime, ForeignKey, Text, UniqueConstraint
from sqlalchemy.orm import relationship
from database import Base


class District(Base):
    """Участок и его коэффициент."""
    __tablename__ = "districts"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, unique=True, nullable=False)   # Название участка
    coeff = Column(Float, nullable=False, default=1.0)    # Коэфф. участка


class User(Base):
    """Сотрудник полиции или начальник отдела."""
    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    username = Column(String, unique=True, nullable=False, index=True)
    password_hash = Column(String, nullable=False)
    full_name = Column(String, nullable=False)            # ФИО сотрудника
    rank = Column(String, default="")                     # Звание
    role = Column(String, nullable=False, default="officer")  # officer | head
    district_id = Column(Integer, ForeignKey("districts.id"), nullable=True)

    district = relationship("District")


class KpiCatalog(Base):
    """Справочник показателей KPI."""
    __tablename__ = "kpi_catalog"

    id = Column(Integer, primary_key=True, index=True)
    code = Column(String, unique=True, nullable=False)    # Код KPI
    indicator = Column(String, nullable=False)            # Показатель
    block = Column(String, nullable=False)                # Блок
    subblock = Column(String, default="")                 # Подблок
    npa = Column(String, default="")                      # НПА/статья
    points_per_unit = Column(Float, nullable=False)       # Балл за 1 ед.
    unit = Column(String, default="ед.")                  # Единица измерения
    source_url = Column(String, default="")               # URL источника
    is_active = Column(Boolean, nullable=False, default=True)


class KpiTarget(Base):
    """Плановый KPI сотрудника на месяц."""
    __tablename__ = "kpi_targets"
    __table_args__ = (
        UniqueConstraint("month", "officer_id", "catalog_id", name="uq_kpi_targets_month_officer_catalog"),
    )

    id = Column(Integer, primary_key=True, index=True)
    month = Column(String, nullable=False)                # Месяц (YYYY-MM)
    officer_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    catalog_id = Column(Integer, ForeignKey("kpi_catalog.id"), nullable=False)
    target_quantity = Column(Float, nullable=False, default=0)

    officer = relationship("User")
    catalog = relationship("KpiCatalog")


class KpiEntry(Base):
    """Запись о выполненной работе сотрудника."""
    __tablename__ = "kpi_entries"

    id = Column(Integer, primary_key=True, index=True)
    date = Column(Date, nullable=False)                   # Дата
    month = Column(String, nullable=False)                # Месяц (YYYY-MM)

    officer_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    catalog_id = Column(Integer, ForeignKey("kpi_catalog.id"), nullable=False)

    quantity = Column(Float, nullable=False, default=1)   # Количество единиц
    points_per_unit = Column(Float, nullable=False)       # Балл за 1 ед. (снимок)
    coeff = Column(Float, nullable=False, default=1.0)    # Коэфф. участка (снимок)
    raw_score = Column(Float, nullable=False, default=0)  # Сырые баллы
    final_score = Column(Float, nullable=False, default=0)  # Итоговые баллы

    status = Column(String, nullable=False, default="pending")  # pending|confirmed|rejected
    comments = Column(Text, default="")                   # Comments (сотрудник)
    head_comment = Column(Text, default="")               # Резолюция начальника
    attachment_name = Column(String, default="")          # Исходное имя файла
    attachment_storage_name = Column(String, default="")  # Безопасное имя на диске
    attachment_content_type = Column(String, default="")
    attachment_size = Column(Integer, nullable=False, default=0)

    officer = relationship("User")
    catalog = relationship("KpiCatalog")


class AppNotification(Base):
    """Персональное системное уведомление пользователя."""
    __tablename__ = "app_notifications"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    kind = Column(String, nullable=False)
    title = Column(String, nullable=False)
    message = Column(Text, nullable=False)
    link = Column(String, nullable=False, default="")
    is_read = Column(Boolean, nullable=False, default=False, index=True)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow, index=True)

    user = relationship("User")


class DirectMessage(Base):
    """Личное сообщение между начальником и сотрудником."""
    __tablename__ = "direct_messages"

    id = Column(Integer, primary_key=True, index=True)
    sender_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    recipient_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    entry_id = Column(Integer, ForeignKey("kpi_entries.id"), nullable=True, index=True)
    body = Column(Text, nullable=False)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow, index=True)
    read_at = Column(DateTime, nullable=True)
    deleted_at = Column(DateTime, nullable=True)

    sender = relationship("User", foreign_keys=[sender_id])
    recipient = relationship("User", foreign_keys=[recipient_id])
    entry = relationship("KpiEntry")
